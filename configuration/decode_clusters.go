package configuration

import (
	"reflect"
	"regexp"
	"strconv"

	"github.com/fabric8-services/fabric8-cluster/cluster/repository"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func decodeClusters(data interface{}) ([]repository.Cluster, error) {
	var clusters []repository.Cluster
	metadata := mapstructure.Metadata{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: &metadata,
		Result:   &clusters,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode clusters from config")
	}
	err = decoder.Decode(data)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode clusters from config")
	}
	// assign default values
	defaultValues, err := getDefaultValues(reflect.TypeOf(repository.Cluster{}))
	if err != nil {
		return nil, errors.Wrap(err, "unable to assign default values on config clusters")
	}
	log.WithField("default_values", defaultValues).Debug("applying default values if needed")
	log.WithField("used_keys", metadata.Keys).Debug("used keys while decoding")
	assignedFields, err := getAssignedFields(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "unable to assign default values on config clusters")
	}
	// now, check if some default values need to be set
	for f, d := range defaultValues {
		for i, c := range clusters {
			log.WithField("cluster#", i).WithField("field", f).Debug("checking if field was assigned")
			if _, assigned := assignedFields[key{idx: i, field: f}]; !assigned {
				log.WithField("cluster#", i).WithField("mapstructure_field", f).Debug("field was not assigned")
				if reflect.ValueOf(&c).Elem().FieldByName(d.name).CanSet() {
					log.WithField("cluster#", i).
						WithField("field_name", d.name).
						WithField("default_value", d.value).
						WithField("can_set", reflect.ValueOf(c).FieldByName(f).CanSet()).
						Info("applying default value...")
					field := reflect.ValueOf(&c).Elem().FieldByName(d.name)
					switch field.Kind() {
					case reflect.Bool:
						field.SetBool(d.value.(bool))
					case reflect.String:
						field.SetString(d.value.(string))
					default:
						return nil, errors.Errorf("unable to assign default values on config clusters: unsupported kind of field: '%s'", field.Kind().String())
					}
					log.WithField("result", c).Debug("applied default value")
				}
			}
			clusters[i] = c
		}
	}

	return clusters, nil
}

// defaultValue a struct to associated a field name with its default value.
type defaultValue struct {
	name  string
	value interface{}
}

// getDefaultValues analyzes the fields in the given type, returning a map with the value of the `default` tag indexed by
// the `mapstructure` tag value.
// Eg: defaultValues["service-account-token-encrypted"] = defaultValue{name:"SATokenEncrypted", value:true}
func getDefaultValues(t reflect.Type) (map[string]defaultValue, error) {
	// check if the receiver struct (here, `repository.Cluster`) has any field tagged with `optional="true" default="..."`
	// and apply the default value accordingdly
	defaultValues := map[string]defaultValue{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		d := f.Tag.Get("default")
		if d != "" {
			switch f.Type.Kind() {
			case reflect.Bool:
				b, err := strconv.ParseBool(d)
				if err == nil {
					defaultValues[f.Tag.Get("mapstructure")] = defaultValue{
						name:  f.Name,
						value: b,
					}
				} else {
					return nil, errors.Wrapf(err, "unable to retrieve default values for type '%s'", t.Name())
				}
			case reflect.String:
				defaultValues[f.Tag.Get("mapstructure")] = defaultValue{
					name:  f.Name,
					value: d,
				}
			default:
				return nil, errors.Errorf("unable to decode clusters from config: unsupported type of field with default: %s", f.Type.String())
			}
		}
	}
	return defaultValues, nil
}

// parse decoder metadata to retrieve the list of all fields that were assigned in the resulting `[]repository.Cluster`
type key struct {
	idx   int
	field string
}

// getAssignedFields analyses the metadata to find which fields were assigned for each element:
// eg: `[0].app-dns` will give `assignedFields[key{idx:0, field:"app=dns"] = true`
// Note that the `true` value associated with the key in the map does not really matter,
// what really matter here is that we have a map to quickly lookup all assigned fields for each element
func getAssignedFields(metadata mapstructure.Metadata) (map[key]interface{}, error) {
	assignedFields := map[key]interface{}{}
	for _, k := range metadata.Keys {
		i, f, err := parseKey(k)
		if err != nil {
			return nil, errors.Wrap(err, "unable to decode clusters from config")
		}
		if i == -1 {
			continue // move to next key
		}
		log.WithField("cluster#", i).WithField("used_field", f).Debug("field was assigned")
		assignedFields[key{idx: i, field: f}] = true
	}
	return assignedFields, nil
}

// parseKey parses the name of an assigned field. Eg:  `[0].app-dns` will return `idx:0, field:"app=dns"`
func parseKey(k string) (int, string, error) {
	r, err := regexp.Compile(`\[(\d+)\]\.(.+)`) // matches things like `[0].app-dns`
	if err != nil {
		return -1, "", errors.Wrap(err, "unable to parse field assigned during decoding")
	}
	if !r.MatchString(k) { // ignore keys such as `[0]`
		return -1, "", nil
	}
	s := r.FindStringSubmatch(k)
	idx, err := strconv.Atoi(s[1])
	if err != nil {
		return -1, "", errors.Errorf("unable to parse field assigned during decoding: '%s' is not an integer", s[1])
	}
	field := s[2]
	return idx, field, nil
}
