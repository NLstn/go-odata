package handlers

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// navigationCreate carries the implicit relationship into the normal create transaction.
type navigationCreate struct {
	parent       interface{}
	propertyName string
	relation     *schema.Relationship
}

func (h *EntityHandler) handlePostNavigationProperty(w http.ResponseWriter, r *http.Request, entityKey, navigationProperty string) {
	name, key := h.parseNavigationPropertyWithKey(navigationProperty)
	navProp := h.findNavigationProperty(name)
	if navProp == nil {
		WriteError(w, r, http.StatusNotFound, "Navigation property not found", navigationProperty)
		return
	}
	if !navProp.NavigationIsArray || key != "" {
		WriteMethodNotAllowed(w, r, "GET, HEAD, OPTIONS", ErrMsgMethodNotAllowed, "POST requires a collection navigation URL")
		return
	}
	targetMetadata, err := h.getTargetMetadata(navProp.NavigationTarget)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, ErrMsgInternalError, err.Error())
		return
	}
	parent, err := h.verifyAndFetchParentEntity(w, r, entityKey)
	if err != nil {
		return
	}
	stmt := &gorm.Statement{DB: h.db}
	if err := stmt.Parse(parent); err != nil {
		WriteError(w, r, http.StatusInternalServerError, ErrMsgInternalError, err.Error())
		return
	}
	relation := stmt.Schema.Relationships.Relations[navProp.Name]
	if relation == nil {
		WriteError(w, r, http.StatusInternalServerError, ErrMsgInternalError, "Navigation relationship is not configured")
		return
	}
	target := h.entityHandlers[targetMetadata.EntitySetName]
	if target == nil {
		target = NewEntityHandler(h.db, targetMetadata, h.logger)
		target.SetEntitiesMetadata(h.entitiesMetadata)
		target.SetNamespace(h.namespace)
		target.SetPolicy(h.policy)
		target.SetDeltaTracker(h.tracker)
		target.SetKeyGeneratorResolver(h.keyGeneratorResolver)
		target.observability = h.observability
	}
	if target.isMethodDisabled(http.MethodPost) {
		WriteMethodNotAllowed(w, r, "GET, HEAD, OPTIONS", ErrMsgMethodNotAllowed, "POST is not allowed for the target entity")
		return
	}
	target.handlePostEntityWithNavigation(w, r, &navigationCreate{parent: parent, propertyName: navProp.Name, relation: relation})
	if w.Header().Get("Location") != "" {
		h.invalidateCache()
	}
}

func (n *navigationCreate) supplyDependentProperties(ctx context.Context, data map[string]interface{}, target *metadata.EntityMetadata) {
	if n.relation.Type == schema.Many2Many {
		return
	}
	for _, ref := range n.relation.References {
		if !ref.OwnPrimaryKey || ref.PrimaryKey == nil {
			continue
		}
		value, _ := ref.PrimaryKey.ValueOf(ctx, reflect.ValueOf(n.parent))
		if prop := target.FindProperty(ref.ForeignKey.Name); prop != nil {
			data[prop.JsonName] = value
			if _, exists := data[prop.Name]; exists {
				data[prop.Name] = value
			}
		}
	}
}

// initialize supplies dependent properties before validation and hooks. Direct
// values for constrained properties are ignored; contradictory explicit bindings fail.
func (n *navigationCreate) initialize(ctx context.Context, entity interface{}, data map[string]interface{}, target *metadata.EntityMetadata) error {
	if n.relation.Type == schema.Many2Many {
		return nil // The join table is populated after the entity is saved.
	}
	parentValue := reflect.ValueOf(n.parent)
	entityValue := reflect.ValueOf(entity)
	for _, ref := range n.relation.References {
		if !ref.OwnPrimaryKey || ref.PrimaryKey == nil {
			continue
		}
		value, _ := ref.PrimaryKey.ValueOf(ctx, parentValue)
		for name := range data {
			if !strings.HasSuffix(name, "@odata.bind") {
				continue
			}
			prop := target.FindProperty(strings.TrimSuffix(name, "@odata.bind"))
			if prop == nil {
				continue
			}
			if _, constrained := prop.ReferentialConstraints[ref.ForeignKey.Name]; constrained {
				boundValue, _ := ref.ForeignKey.ValueOf(ctx, entityValue)
				if !reflect.DeepEqual(indirectNavigationValue(boundValue), indirectNavigationValue(value)) {
					return fmt.Errorf("binding %s contradicts the navigation URL", name)
				}
			}
		}
		if err := ref.ForeignKey.Set(ctx, entityValue, value); err != nil {
			return err
		}
		if prop := target.FindProperty(ref.ForeignKey.Name); prop != nil {
			data[prop.JsonName] = value
			if _, exists := data[prop.Name]; exists {
				data[prop.Name] = value
			}
		}
	}
	return nil
}

func indirectNavigationValue(value interface{}) interface{} {
	v := reflect.ValueOf(value)
	for v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) {
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil
	}
	return v.Interface()
}
