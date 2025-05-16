package discord

import (
	"fmt"
	"strings"
)

const ComponentIDPrefix = "o:"

type ComponentIDSource string
type ComponentIDAction string

type ComponentID struct {
	// Source is the original context, ie: "settings"
	Source ComponentIDSource

	// Action is what the component should do, ie: "enable_asr"
	Action ComponentIDAction
}

var (
	ErrComponentIDInvalidPrefix = fmt.Errorf("invalid component id prefix")
	ErrComponentIDInvalidParts  = fmt.Errorf("incorrect number of parts in component id")
)

func ParseComponentID(id string) (*ComponentID, error) {
	id, found := strings.CutPrefix(id, ComponentIDPrefix)
	if !found {
		return nil, ErrComponentIDInvalidPrefix
	}

	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		return nil, ErrComponentIDInvalidParts
	}

	parsedID := &ComponentID{
		Source: ComponentIDSource(parts[0]),
		Action: ComponentIDAction(parts[1]),
	}

	return parsedID, nil
}

func (c *ComponentID) String() string {
	return ComponentIDPrefix + strings.Join([]string{
		string(c.Source),
		string(c.Action),
	}, ":")
}

func ComponentIDString(source ComponentIDSource, action ComponentIDAction) string {
	componentID := &ComponentID{
		Source: source,
		Action: action,
	}
	return componentID.String()
}
