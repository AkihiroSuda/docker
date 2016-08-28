package daemon

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/container"
	"github.com/docker/docker/volume"
	"github.com/docker/docker/volume/introspection"
)

const (
	introspectionRegularFilePerm = 0644
)

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
func (daemon *Daemon) updateIntrospection(c *container.Container) error {
	conn := getIntrospectionConnector(c)
	if conn == nil {
		return nil
	}
	ctx := daemon.introspectRuntimeContext(c)
	return updateIntrospection(conn, "", reflect.ValueOf(ctx))
}

func updateIntrospection(conn volume.ContainerVolumeConnector,
	path string, val reflect.Value) error {
	switch val.Kind() {
	case reflect.Struct:
		return updateIntrospectionStruct(conn, path, val)
	case reflect.Int:
		return updateIntrospectionInt(conn, path, val)
	case reflect.String:
		return updateIntrospectionString(conn, path, val)
	case reflect.Map:
		return updateIntrospectionMap(conn, path, val)
	case reflect.Ptr:
		if val.IsNil() {
			return nil
		}
		return updateIntrospection(conn, path, val.Elem())
	default:
		return fmt.Errorf("unsupported kind: %v", val.Kind())
	}
}

func updateIntrospectionStruct(conn volume.ContainerVolumeConnector,
	path string, val reflect.Value) error {
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected reflect.Struct, got %v", val.Kind())
	}
	typ := val.Type()
	fields := val.NumField()
	for i := 0; i < fields; i++ {
		// **RFC** we call ToLower for the naming convention
		fieldPath := strings.ToLower(filepath.Join(path, typ.Field(i).Name))
		fieldVal := val.Field(i)
		if err := updateIntrospection(conn, fieldPath, fieldVal); err != nil {
			return err
		}
	}
	return nil
}

func updateIntrospectionInt(conn volume.ContainerVolumeConnector,
	path string, val reflect.Value) error {
	if val.Kind() != reflect.Int {
		return fmt.Errorf("expected reflect.Int, got %v", val.Kind())
	}
	d := val.Interface().(int)
	return conn.Update(path,
		[]byte(fmt.Sprintf("%d\n", d)),
		introspectionRegularFilePerm)
}

func updateIntrospectionString(conn volume.ContainerVolumeConnector,
	path string, val reflect.Value) error {
	if val.Kind() != reflect.String {
		return fmt.Errorf("expected reflect.String, got %v", val.Kind())
	}
	s := val.Interface().(string)
	if len(s) > 0 {
		s += "\n"
	}
	return conn.Update(path,
		[]byte(s),
		introspectionRegularFilePerm)
}

func validateIntrospectionMapKeyString(s string) error {
	banned := "/\\:"
	if strings.ContainsAny(s, banned) {
		return fmt.Errorf("invalid map key string %s: should not contain %s)",
			s, banned)
	}
	return nil
}

func updateIntrospectionMap(conn volume.ContainerVolumeConnector,
	path string, val reflect.Value) error {
	if val.Kind() != reflect.Map {
		return fmt.Errorf("expected reflect.Map, got %v", val.Kind())
	}
	for _, mapK := range val.MapKeys() {
		if mapK.Kind() != reflect.String {
			return fmt.Errorf("expected reflect.String for map key, got %v", mapK.Kind())
		}
		mapV := val.MapIndex(mapK)
		if mapV.Kind() != reflect.String {
			return fmt.Errorf("expected reflect.String for map value, got %v", mapV.Kind())
		}
		key := mapK.Interface().(string)
		if err := validateIntrospectionMapKeyString(key); err != nil {
			// err occurs typically when key contains '/'
			logrus.Warn(err)
			continue
		}
		// we don't call strings.ToLower() and keep the original key string here
		keyPath := filepath.Join(path, key)
		if err := updateIntrospectionString(conn, keyPath, mapV); err != nil {
			return err
		}
	}
	return nil
}

// getIntrospectionConnector returns a connector for interaction between the daemon
// and the introspection volume. Future version may return non-volume connector.
func getIntrospectionConnector(c *container.Container) volume.ContainerVolumeConnector {
	for _, m := range c.MountPoints {
		if v, ok := m.Volume.(*introspection.Volume); ok {
			return v.Connector()
		}
	}
	return nil
}
