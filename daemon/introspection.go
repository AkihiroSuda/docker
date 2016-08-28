package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/container"
	"github.com/docker/docker/pkg/ioutils"
)

const (
	introspectionRegularFilePerm = 0644
)

type introspectionOptions struct {
	scopes []string
}

// updateIntrospection updates the actual content of the inspection volume.
//
// The layout is defined as the RuntimeContext structure.
//
// Format convention for supported types:
//   - struct:       the field name is used for the directory name
//   - int:          "%d\n"
//   - string:       "%s\n" for non-empty string, "" for empty string
//   - map[string]..: the key string is used for the filename
//
// **RFC**: do we need "\n" at terminal?
// Note: For an empty string, an empty file (without "\n" at terminal) is created
func (daemon *Daemon) updateIntrospection(c *container.Container, opts *introspectionOptions) error {
	conn := getIntrospectionConnector(c)
	if conn == nil {
		return nil
	}
	ctx := daemon.introspectRuntimeContext(c)
	return updateIntrospection(conn, opts.scopes, "", "", reflect.ValueOf(ctx))
}

// updateIntrospection recursively updates introspection.
// scope: e.g. ".container.labels"
// path:  e.g. "\container\labels\org.example.foo.bar"
func updateIntrospection(conn introspectionConnector, scopes []string,
	scope, path string, val reflect.Value) error {
	switch val.Kind() {
	case reflect.Struct:
		return updateIntrospectionStruct(conn, scopes, scope, path, val)
	case reflect.Int:
		return updateIntrospectionInt(conn, scopes, scope, path, val)
	case reflect.String:
		return updateIntrospectionString(conn, scopes, scope, path, val)
	case reflect.Map:
		return updateIntrospectionMap(conn, scopes, scope, path, val)
	case reflect.Ptr:
		if val.IsNil() {
			return nil
		}
		return updateIntrospection(conn, scopes, scope, path, val.Elem())
	default:
		return fmt.Errorf("unsupported kind: %v", val.Kind())
	}
}

func updateIntrospectionStruct(conn introspectionConnector, scopes []string,
	scope, path string, val reflect.Value) error {
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected reflect.Struct, got %v", val.Kind())
	}
	typ := val.Type()
	fields := val.NumField()
	for i := 0; i < fields; i++ {
		// **RFC** we call ToLower for the naming convention
		fieldName := strings.ToLower(typ.Field(i).Name)
		fieldScope := scope + "." + fieldName
		fieldPath := filepath.Join(path, fieldName)
		fieldVal := val.Field(i)
		if err := updateIntrospection(conn, scopes, fieldScope, fieldPath, fieldVal); err != nil {
			return err
		}
	}
	return nil
}

func updateIntrospectionInt(conn introspectionConnector, scopes []string,
	scope, path string, val reflect.Value) error {
	if val.Kind() != reflect.Int {
		return fmt.Errorf("expected reflect.Int, got %v", val.Kind())
	}
	d := val.Interface().(int)
	return callConnectorUpdate(conn, scopes, scope, path, []byte(fmt.Sprintf("%d\n", d)))
}

func updateIntrospectionString(conn introspectionConnector, scopes []string,
	scope, path string, val reflect.Value) error {
	if val.Kind() != reflect.String {
		return fmt.Errorf("expected reflect.String, got %v", val.Kind())
	}
	s := val.Interface().(string)
	if len(s) > 0 {
		s += "\n"
	}
	return callConnectorUpdate(conn, scopes, scope, path, []byte(s))
}

func callConnectorUpdate(conn introspectionConnector, scopes []string, scope, path string, content []byte) error {
	if inScope(scope, scopes) {
		return conn.Update(path, content, introspectionRegularFilePerm)
	}
	return nil
}

func validateIntrospectionMapKeyString(s string) error {
	banned := "/\\:"
	if strings.ContainsAny(s, banned) {
		return fmt.Errorf("invalid map key string %s: should not contain %s)",
			s, banned)
	}
	return nil
}

func updateIntrospectionMap(conn introspectionConnector, scopes []string,
	scope, path string, val reflect.Value) error {
	if val.Kind() != reflect.Map {
		return fmt.Errorf("expected reflect.Map, got %v", val.Kind())
	}
	for _, mapK := range val.MapKeys() {
		if mapK.Kind() != reflect.String {
			return fmt.Errorf("expected reflect.String for map key, got %v", mapK.Kind())
		}
		key := mapK.Interface().(string)
		if err := validateIntrospectionMapKeyString(key); err != nil {
			// err occurs typically when key contains '/'
			logrus.Warn(err)
			continue
		}
		// e.g. mapScope=".container.labels", keyPath="\container\labels\org.example.foo.bar"
		mapScope := scope
		// we don't call strings.ToLower() and keep the original key string here
		keyPath := filepath.Join(path, key)
		mapV := val.MapIndex(mapK)
		if err := updateIntrospectionString(conn, scopes, mapScope, keyPath, mapV); err != nil {
			return err
		}
	}
	return nil
}

type introspectionConnector interface {
	Update(path string, content []byte, perm os.FileMode) error
}

type fsIntrospectionConnector struct {
	dir string
}

func (conn *fsIntrospectionConnector) Update(path string, content []byte, perm os.FileMode) error {
	realPath := filepath.Join(conn.dir, path)
	if err := os.MkdirAll(filepath.Dir(realPath), 0755); err != nil {
		return err
	}
	if content == nil {
		return os.RemoveAll(realPath)
	}
	return ioutils.AtomicWriteFile(realPath, content, perm)
}

// getIntrospectionConnector returns a connector for interaction between the daemon
// and the introspection filesystem. Future version may return a non-filesystem connector.
func getIntrospectionConnector(c *container.Container) introspectionConnector {
	return &fsIntrospectionConnector{dir: c.IntrospectionDir()}
}

func inScope(scope string, scopes []string) bool {
	for _, s := range scopes {
		if strings.HasPrefix(scope, s) {
			return true
		}
	}
	return false
}
