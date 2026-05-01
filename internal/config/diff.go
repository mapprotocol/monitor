package config

import (
	"reflect"
	"strings"
)

// ChainDiff classifies what a reload should do for each chain in the new
// configuration relative to the old one.
//
//   - Adds:     chains present in new but not in old           (start fresh)
//   - Removes:  names present in old but not in new            (Stop + drop)
//   - Restarts: same name, but a structural field changed      (Stop + Start with new build)
//   - Updates:  same name, only data fields changed            (in-place ApplyHotReloadable)
//
// Structural means the field can't be mutated in place: Endpoint, Network.
// Other immutable fields (Type, Id, KeystorePath, opts.checkHeightCount,
// opts.changeInterval) are filtered out earlier by diffImmutable in the
// reloader, so DiffChains assumes the input is already validated.
type ChainDiff struct {
	Adds     []RawChainConfig
	Removes  []string
	Restarts []RawChainConfig
	Updates  []RawChainConfig
}

// DiffChains classifies each chain transition between old and new.
// Inputs are matched by lowercase Name.
func DiffChains(old, new []RawChainConfig) ChainDiff {
	oldByName := indexByName(old)
	newByName := indexByName(new)

	var d ChainDiff

	// Iterate new -> classify each as Add / Restart / Update.
	for _, nc := range new {
		oc, exists := oldByName[strings.ToLower(nc.Name)]
		if !exists {
			d.Adds = append(d.Adds, nc)
			continue
		}
		if structuralChanged(oc, nc) {
			d.Restarts = append(d.Restarts, nc)
			continue
		}
		if dataChanged(oc, nc) {
			d.Updates = append(d.Updates, nc)
		}
	}

	// Anything in old but not in new -> Remove.
	for _, oc := range old {
		if _, exists := newByName[strings.ToLower(oc.Name)]; !exists {
			d.Removes = append(d.Removes, oc.Name)
		}
	}
	return d
}

// structuralChanged reports whether oc -> nc requires tearing down the
// chain (its Connection) and starting a fresh one.
func structuralChanged(oc, nc RawChainConfig) bool {
	return oc.Endpoint != nc.Endpoint || oc.Network != nc.Network
}

// dataChanged reports whether any hot-reloadable field differs. We compare
// the cheap scalar-and-slice fields with reflect.DeepEqual to keep this
// honest as new fields are added.
func dataChanged(oc, nc RawChainConfig) bool {
	if oc.From != nc.From {
		return true
	}
	if !reflect.DeepEqual(oc.Users, nc.Users) {
		return true
	}
	if !reflect.DeepEqual(oc.ContractToken, nc.ContractToken) {
		return true
	}
	if !reflect.DeepEqual(oc.Energies, nc.Energies) {
		return true
	}
	if !reflect.DeepEqual(oc.Tss, nc.Tss) {
		return true
	}
	// opts: ignore the immutable keys (checkHeightCount, changeInterval)
	// — those are caught earlier and would never reach here, but be safe.
	return !optsHotEqual(oc.Opts, nc.Opts)
}

func optsHotEqual(a, b map[string]string) bool {
	keys := map[string]struct{}{}
	for k := range a {
		keys[k] = struct{}{}
	}
	for k := range b {
		keys[k] = struct{}{}
	}
	for k := range keys {
		if k == CheckHeightCount || k == ChangeInterval {
			continue
		}
		if a[k] != b[k] {
			return false
		}
	}
	return true
}
