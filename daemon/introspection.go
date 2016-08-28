package daemon

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/docker/docker/container"
	"github.com/docker/docker/volume"
	"github.com/docker/docker/volume/introspection"
)

const (
	introspectionBaseDir         = "v1"
	introspectionRegularFilePerm = 0644
)

// updateIntrospection updates the actual content of the inspection volume.
//
// The layout is defined as the RuntimeContext structure.
//
// Format convention for supported types:
//   - struct: (created as a directory)
//   - int (typically Slot): "%d\n"
//   - string (typically ID, Name): "%s\n" (empty file **created** if s="")
//   - map[string]string (typically Labels): "%s=%s\n,%s=%s\n,.." (ditto)
//
// Note that even for empty fields, files are always created.
// So user don't need to consider whether the file exists.
// It also simplifies the implementation of atomic update of files.
//
// TODO: support dynamic label update
func (daemon *Daemon) updateIntrospection(c *container.Container) error {
	conn := getIntrospectionConnector(c)
	if conn == nil {
		return nil
	}
	ctx := daemon.introspectRuntimeContext(c)
	return updateIntrospectionStruct(conn,
		introspectionBaseDir,
		reflect.TypeOf(ctx), reflect.ValueOf(ctx))
}

func updateIntrospectionStruct(conn volume.ContainerVolumeConnector,
	cwd string,
	typ reflect.Type,
	val reflect.Value) error {
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("expected reflect.Struct, got %v", val.Kind())
	}
	fields := val.NumField()
	for i := 0; i < fields; i++ {
		tField, vField := typ.Field(i), val.Field(i)
		path := strings.ToLower(filepath.Join(cwd, tField.Name))
		switch vField.Kind() {
		case reflect.Struct:
			if err := updateIntrospectionStruct(conn,
				path,
				vField.Type(),
				vField); err != nil {
				return err
			}
		case reflect.Int:
			if err := updateIntrospectionInt(conn,
				path,
				vField); err != nil {
				return err
			}
		case reflect.String:
			if err := updateIntrospectionString(conn,
				path,
				vField); err != nil {
				return err
			}
		case reflect.Map:
			if err := updateIntrospectionMap(conn,
				path,
				vField); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported kind: %v", vField.Kind())
		}
	}
	return nil
}

func updateIntrospectionInt(conn volume.ContainerVolumeConnector,
	path string,
	val reflect.Value) error {
	if val.Kind() != reflect.Int {
		return fmt.Errorf("expected reflect.Int, got %v", val.Kind())
	}
	d := val.Interface().(int)
	return conn.Update(path,
		[]byte(fmt.Sprintf("%d\n", d)),
		introspectionRegularFilePerm)
}

func updateIntrospectionString(conn volume.ContainerVolumeConnector,
	path string,
	val reflect.Value) error {
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

func updateIntrospectionMap(conn volume.ContainerVolumeConnector,
	path string,
	val reflect.Value) error {
	if val.Kind() != reflect.Map {
		return fmt.Errorf("expected reflect.Map, got %v", val.Kind())
	}
	s := ""
	for _, mapK := range val.MapKeys() {
		if mapK.Kind() != reflect.String {
			return fmt.Errorf("expected reflect.String for map key, got %v", mapK.Kind())
		}
		mapV := val.MapIndex(mapK)
		if mapV.Kind() != reflect.String {
			return fmt.Errorf("expected reflect.String for map value, got %v", mapV.Kind())
		}
		s += fmt.Sprintf("%s=%s\n",
			mapK.Interface().(string),
			mapV.Interface().(string))
	}
	return conn.Update(path,
		[]byte(s),
		introspectionRegularFilePerm)
}

func getIntrospectionConnector(c *container.Container) volume.ContainerVolumeConnector {
	for _, m := range c.MountPoints {
		if v, ok := m.Volume.(*introspection.Volume); ok {
			return v.Connector()
		}
	}
	return nil
}
