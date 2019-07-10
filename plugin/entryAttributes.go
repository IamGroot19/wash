package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Benchkram/errz"
	"github.com/puppetlabs/wash/munge"
)

// JSONObject is a typedef to a map[string]interface{}
// object.
type JSONObject = map[string]interface{}

// ToJSONObject serializes v to a JSON object. It will panic
// if the serialization fails.
func ToJSONObject(v interface{}) JSONObject {
	if obj, ok := v.(JSONObject); ok {
		return obj
	}
	var err error
	var inrec []byte
	if arr, ok := v.([]byte); ok {
		inrec = arr
	} else {
		if inrec, err = json.Marshal(v); err != nil {
			// Internal error if we can't marshal an object
			panic(err)
		}
	}
	var obj JSONObject
	// Internal error if not a JSON object
	errz.Fatal(json.Unmarshal(inrec, &obj))
	return obj
}

/*
EntryAttributes represents an entry's attributes. We use a struct
instead of a map for efficient memory allocation/deallocation,
which is needed to make Parent#List fast.

Each of the setters supports the builder pattern, which enables you
to do something like

	attr := plugin.EntryAttributes{}
	attr.
		SetCtime(ctime).
		SetMtime(mtime).
		SetMeta(meta)
	entry.SetAttributes(attr)
*/
type EntryAttributes struct {
	atime   time.Time
	mtime   time.Time
	ctime   time.Time
	mode    os.FileMode
	hasMode bool
	size    uint64
	hasSize bool
	meta    JSONObject
}

// We can't just export EntryAttributes' fields because there's no way
// to determine if an arbitrary entry has e.g. a 'size' attribute from
// the size value alone (since 0-size is valid). That's why we have the
// separate has* fields, and that's why those attributes need their own
// setters. However, it's a bit weird to have setters for some fields
// and not have setters for others (e.g. we could export atime, mtime,
// ctime b/c we know that an entry has atime/mtime/ctime if their value
// isn't the zero-time). It also increases the chance that a plugin author
// could inadvertantly forget to call the `size`/`mode` attribute setter
// when creating their attributes and instead set those values in the
// constructor (via something like EntryAttributes{Ctime: time.Now(), Size: 15}).
// The latter's bad b/c Wash would think the entry didn't have a size attribute
// (since hasSize is false).
//
// Thus, although these getters/setters/Has* methods are annoying, they're
// the best way to maintain a clean and consistent interface for EntryAttributes
// while minimizing plugin author error.

// HasAtime returns true if the entry has a last access time
func (a *EntryAttributes) HasAtime() bool {
	return !a.atime.IsZero()
}

// Atime returns the entry's last access time
func (a *EntryAttributes) Atime() time.Time {
	return a.atime
}

// SetAtime sets the entry's last access time
func (a *EntryAttributes) SetAtime(atime time.Time) *EntryAttributes {
	a.atime = atime
	return a
}

// HasMtime returns true if the entry has a last modified time
func (a *EntryAttributes) HasMtime() bool {
	return !a.mtime.IsZero()
}

// Mtime returns the entry's last modified time
func (a *EntryAttributes) Mtime() time.Time {
	return a.mtime
}

// SetMtime sets the entry's last modified time
func (a *EntryAttributes) SetMtime(mtime time.Time) *EntryAttributes {
	a.mtime = mtime
	return a
}

// HasCtime returns true if the entry has a creation time
func (a *EntryAttributes) HasCtime() bool {
	return !a.ctime.IsZero()
}

// Ctime returns the entry's creation time
func (a *EntryAttributes) Ctime() time.Time {
	return a.ctime
}

// SetCtime sets the entry's creation time
func (a *EntryAttributes) SetCtime(ctime time.Time) *EntryAttributes {
	a.ctime = ctime
	return a
}

// HasMode returns true if the entry has a mode
func (a *EntryAttributes) HasMode() bool {
	return a.hasMode
}

// Mode returns the entry's mode
func (a *EntryAttributes) Mode() os.FileMode {
	return a.mode
}

// SetMode sets the entry's mode
func (a *EntryAttributes) SetMode(mode os.FileMode) *EntryAttributes {
	a.mode = mode
	a.hasMode = true
	return a
}

// HasSize returns true if the entry has a size
func (a *EntryAttributes) HasSize() bool {
	return a.hasSize
}

// Size returns the entry's Size
func (a *EntryAttributes) Size() uint64 {
	return a.size
}

// SetSize sets the entry's size
func (a *EntryAttributes) SetSize(size uint64) *EntryAttributes {
	a.size = size
	a.hasSize = true
	return a
}

// Meta returns the entry's meta attribute. If a.SetMeta(obj) was called,
// then this returns obj serialized to JSONObject. Otherwise, it returns
// a.ToMap(false).
//
// NOTE: The meta attribute is a subset of the entry's full metadata, which
// is what e.Metadata returns. It is typically provided by the plugin API's
// List endpoint.
func (a *EntryAttributes) Meta() JSONObject {
	if a.meta == nil {
		return a.ToMap(false)
	}

	return a.meta
}

// SetMeta sets the entry's meta attribute to obj. This is typically the
// raw object that's returned by the plugin API's List endpoint, or a wrapper
// that includes the raw object + some additional information. For example, if
// the entry represents a Docker container, then obj would be a Container struct.
// If the entry represents a Docker volume, then obj would be a Volume struct.
//
// SetMeta will panic if obj does not serialize to a JSON object.
func (a *EntryAttributes) SetMeta(obj interface{}) *EntryAttributes {
	a.meta = ToJSONObject(obj)
	return a
}

// ToMap converts the entry's attributes to a map, which makes it easier to write
// generic code on them.
func (a *EntryAttributes) ToMap(includeMeta bool) map[string]interface{} {
	mp := make(map[string]interface{})
	if a.HasAtime() {
		mp["atime"] = a.Atime()
	}
	if a.HasMtime() {
		mp["mtime"] = a.Mtime()
	}
	if a.HasCtime() {
		mp["ctime"] = a.Ctime()
	}
	if a.HasMode() {
		// The mode string representation is the only portable representation. FileMode uses its own
		// definitions for type bits, not those in http://man7.org/linux/man-pages/man7/inode.7.html.
		mp["mode"] = a.Mode().String()
	}
	if a.HasSize() {
		mp["size"] = a.Size()
	}
	if includeMeta {
		mp["meta"] = a.Meta()
	}
	return mp
}

// MarshalJSON marshals the entry's attributes to JSON. It takes
// a value receiver so that the attributes are still marshaled
// when they're referenced as interface{} objects. See
// https://stackoverflow.com/a/21394657 for more details.
func (a EntryAttributes) MarshalJSON() ([]byte, error) {
	m := a.ToMap(true)
	// Override Mode to use a re-marshalable representation.
	if a.HasMode() {
		m["mode"] = a.Mode()
	}
	return json.Marshal(m)
}

// UnmarshalJSON unmarshals the entry's attributes from JSON.
func (a *EntryAttributes) UnmarshalJSON(data []byte) error {
	mp := make(map[string]interface{})
	err := json.Unmarshal(data, &mp)
	if err != nil {
		return fmt.Errorf("plugin.EntryAttributes.UnmarshalJSON received a non-JSON object")
	}
	if atime, ok := mp["atime"]; ok {
		t, err := munge.ToTime(atime)
		if err != nil {
			return attrMungeError("atime", err)
		}
		a.SetAtime(t)
	}
	if mtime, ok := mp["mtime"]; ok {
		t, err := munge.ToTime(mtime)
		if err != nil {
			return attrMungeError("mtime", err)
		}
		a.SetMtime(t)
	}
	if ctime, ok := mp["ctime"]; ok {
		t, err := munge.ToTime(ctime)
		if err != nil {
			return attrMungeError("ctime", err)
		}
		a.SetCtime(t)
	}
	if mode, ok := mp["mode"]; ok {
		// Even though os.FileModes are uint32 types, json.Unmarshal unmarshals them as float64.
		// That's ok, because float64 has sufficient precision to represent all uint32 types.
		if raw, ok := mode.(float64); ok {
			a.SetMode(os.FileMode(raw))
		} else {
			return attrMungeError("mode", fmt.Errorf("mode was unexpected type %T: %v", mode, mode))
		}
	}
	if size, ok := mp["size"]; ok {
		sz, err := munge.ToSize(size)
		if err != nil {
			return attrMungeError("size", err)
		}
		a.SetSize(sz)
	}
	if rawMeta, ok := mp["meta"]; ok {
		meta, isObj := rawMeta.(JSONObject)
		if !isObj {
			return fmt.Errorf("meta is not a JSON object")
		}
		a.SetMeta(meta)
	}
	return nil
}

func attrMungeError(name string, err error) error {
	return fmt.Errorf("plugin.EntryAttributes.UnmarshalJSON: could not munge the %v attribute: %v", name, err)
}
