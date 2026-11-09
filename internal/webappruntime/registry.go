package webappruntime

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Options[T comparable] struct {
	RuntimeID string
	GroupKey  string
	AppID     string
	Value     T
}

type Binding[T comparable] struct {
	AppID string
	Value T
}

type entry[T comparable] struct {
	id       string
	groupKey string
	value    T
	apps     map[string]struct{}
}

type Registry[T comparable] struct {
	mu           sync.RWMutex
	byID         map[string]*entry[T]
	byGroup      map[string]string
	runtimeByApp map[string]string
}

func NewRegistry[T comparable]() *Registry[T] {
	return &Registry[T]{
		byID: make(map[string]*entry[T]), byGroup: make(map[string]string), runtimeByApp: make(map[string]string),
	}
}

func (registry *Registry[T]) Register(options Options[T]) error {
	options.RuntimeID = strings.TrimSpace(options.RuntimeID)
	options.GroupKey = strings.TrimSpace(options.GroupKey)
	options.AppID = strings.TrimSpace(options.AppID)
	if options.RuntimeID == "" || options.GroupKey == "" || options.AppID == "" {
		return errors.New("Web App runtime, group and app IDs are required")
	}
	item := &entry[T]{
		id: options.RuntimeID, groupKey: options.GroupKey, value: options.Value,
		apps: map[string]struct{}{options.AppID: {}},
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.byID[item.id]; exists {
		return fmt.Errorf("Web App runtime %q already exists", item.id)
	}
	if _, exists := registry.byGroup[item.groupKey]; exists {
		return fmt.Errorf("Web App runtime group %q already exists", item.groupKey)
	}
	registry.detachAppLocked(options.AppID)
	registry.byID[item.id] = item
	registry.byGroup[item.groupKey] = item.id
	registry.runtimeByApp[options.AppID] = item.id
	return nil
}

func (registry *Registry[T]) Attach(runtimeID string, appID string) error {
	runtimeID = strings.TrimSpace(runtimeID)
	appID = strings.TrimSpace(appID)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	item := registry.byID[runtimeID]
	if item == nil {
		return fmt.Errorf("Web App runtime %q not found", runtimeID)
	}
	if appID == "" {
		return errors.New("Web App id is required")
	}
	registry.detachAppLocked(appID)
	item.apps[appID] = struct{}{}
	registry.runtimeByApp[appID] = runtimeID
	return nil
}

func (registry *Registry[T]) Get(runtimeID string) (T, bool) {
	registry.mu.RLock()
	item := registry.byID[strings.TrimSpace(runtimeID)]
	registry.mu.RUnlock()
	if item == nil {
		var zero T
		return zero, false
	}
	return item.value, true
}

func (registry *Registry[T]) ForApp(appID string) (T, bool) {
	registry.mu.RLock()
	item := registry.byID[registry.runtimeByApp[strings.TrimSpace(appID)]]
	registry.mu.RUnlock()
	if item == nil {
		var zero T
		return zero, false
	}
	return item.value, true
}

func (registry *Registry[T]) FindByGroup(groupKey string) (T, bool) {
	registry.mu.RLock()
	item := registry.byID[registry.byGroup[strings.TrimSpace(groupKey)]]
	registry.mu.RUnlock()
	if item == nil {
		var zero T
		return zero, false
	}
	return item.value, true
}

func (registry *Registry[T]) Bindings() []Binding[T] {
	registry.mu.RLock()
	bindings := make([]Binding[T], 0, len(registry.runtimeByApp))
	for appID, runtimeID := range registry.runtimeByApp {
		if item := registry.byID[runtimeID]; item != nil {
			bindings = append(bindings, Binding[T]{AppID: appID, Value: item.value})
		}
	}
	registry.mu.RUnlock()
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].AppID < bindings[j].AppID })
	return bindings
}

func (registry *Registry[T]) AppIDs(runtimeID string) []string {
	registry.mu.RLock()
	item := registry.byID[strings.TrimSpace(runtimeID)]
	apps := make([]string, 0)
	if item != nil {
		apps = make([]string, 0, len(item.apps))
		for appID := range item.apps {
			apps = append(apps, appID)
		}
	}
	registry.mu.RUnlock()
	sort.Strings(apps)
	return apps
}

func (registry *Registry[T]) DetachApp(appID string) (T, int, bool) {
	registry.mu.Lock()
	item := registry.detachAppLocked(strings.TrimSpace(appID))
	remaining := 0
	if item != nil {
		remaining = len(item.apps)
	}
	registry.mu.Unlock()
	if item == nil {
		var zero T
		return zero, 0, false
	}
	return item.value, remaining, true
}

func (registry *Registry[T]) Remove(runtimeID string, expected T) bool {
	runtimeID = strings.TrimSpace(runtimeID)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	item := registry.byID[runtimeID]
	if item == nil || item.value != expected {
		return false
	}
	delete(registry.byID, runtimeID)
	if registry.byGroup[item.groupKey] == runtimeID {
		delete(registry.byGroup, item.groupKey)
	}
	for appID := range item.apps {
		if registry.runtimeByApp[appID] == runtimeID {
			delete(registry.runtimeByApp, appID)
		}
	}
	return true
}

func (registry *Registry[T]) Values() []T {
	registry.mu.RLock()
	values := make([]T, 0, len(registry.byID))
	for _, item := range registry.byID {
		values = append(values, item.value)
	}
	registry.mu.RUnlock()
	return values
}

func (registry *Registry[T]) detachAppLocked(appID string) *entry[T] {
	runtimeID := registry.runtimeByApp[appID]
	item := registry.byID[runtimeID]
	if item != nil {
		delete(item.apps, appID)
	}
	delete(registry.runtimeByApp, appID)
	return item
}
