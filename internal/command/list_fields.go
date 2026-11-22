package command

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type listField[T any] struct {
	name, header, shortHeader string
	value                     func(T) any
}

type listFields[T any] []listField[T]

func (fields listFields[T]) names() []string {
	names := make([]string, len(fields))
	for i, field := range fields {
		names[i] = field.name
	}
	return names
}

func (fields listFields[T]) selectFields(resource string, names []string) (listFields[T], error) {
	if names == nil {
		return fields, nil
	}
	selected := make(listFields[T], 0, len(names))
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, usage("--fields requires a comma-separated list of field names")
		}
		i := slices.IndexFunc(fields, func(field listField[T]) bool { return field.name == name })
		if i < 0 {
			return nil, usagef("unknown %s field %q (expected one of %s)", resource, name, strings.Join(fields.names(), ", "))
		}
		if seen[name] {
			return nil, usagef("duplicate %s field %q", resource, name)
		}
		seen[name] = true
		selected = append(selected, fields[i])
	}
	return selected, nil
}

func optionalListValue[T any](value *T) any {
	if value == nil {
		return nil
	}
	return *value
}

func writeList[T any](ctx context.Context, a App, records []T, fields listFields[T], format string, options tableOutputOptions) error {
	if format == "json" {
		objects := make([]map[string]any, len(records))
		for i, record := range records {
			object := make(map[string]any, len(fields))
			for _, field := range fields {
				object[field.name] = field.value(record)
			}
			objects[i] = object
		}
		b, err := json.MarshalIndent(objects, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(a.Out, string(b))
		return err
	}

	rows := make([][]string, len(records))
	for i, record := range records {
		row := make([]string, len(fields))
		for j, field := range fields {
			switch value := field.value(record).(type) {
			case nil:
			case float64:
				row[j] = strconv.FormatFloat(value, 'f', -1, 64)
			default:
				row[j] = fmt.Sprint(value)
			}
		}
		rows[i] = row
	}
	if format == "table" {
		headers := make([]string, len(fields))
		shortHeaders := make([]string, len(fields))
		for i, field := range fields {
			headers[i], shortHeaders[i] = field.header, field.shortHeader
		}
		return a.writeTable(ctx, options, headers, rows, shortHeaders)
	}
	_, err := fmt.Fprintln(a.Out, renderDelimited(fields.names(), rows, format, true))
	return err
}
