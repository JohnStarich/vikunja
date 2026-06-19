package models

import (
	"fmt"
	"iter"

	"code.vikunja.io/api/pkg/db"
	"github.com/pkg/errors"
	"xorm.io/builder"
	"xorm.io/xorm"
)

type UrgencyProperty int

const (
	minUrgencyProperty UrgencyProperty = iota // ensure zero-value is invalid to better detect bugs

	UrgencyDueDate
	UrgencyMatchesFilter
	UrgencyPercentDone
	UrgencyPriority

	maxUrgencyProperty
)

func AllUrgencyProperties() iter.Seq[UrgencyProperty] {
	return func(yield func(UrgencyProperty) bool) {
		for property := minUrgencyProperty + 1; property < maxUrgencyProperty; property++ {
			if !yield(property) {
				return
			}
		}
	}
}

func (u UrgencyProperty) name() (string, error) {
	switch u {
	case UrgencyDueDate:
		return "due_date", nil
	case UrgencyMatchesFilter:
		return "matches_filter", nil
	case UrgencyPercentDone:
		return "percent_done", nil
	case UrgencyPriority:
		return "priority", nil
	default:
		return "", fmt.Errorf("invalid urgency property enum value: %d", u)
	}
}

func (u UrgencyProperty) String() string {
	name, err := u.name()
	if err != nil {
		panic(err)
	}
	return name
}

func (u UrgencyProperty) MarshalText() ([]byte, error) {
	name, err := u.name()
	return []byte(name), err
}

func (u *UrgencyProperty) UnmarshalText(b []byte) error {
	name := string(b)
	for property := range AllUrgencyProperties() {
		if name == property.String() {
			*u = property
			return nil
		}
	}
	return fmt.Errorf("unknown urgency property: %q", string(b))
}

type UrgencyWeight struct {
	SavedFilterID int64           `xorm:"not null unique(weight)"`
	Property      string          `xorm:"varchar(32) not null unique(weight)"` // TODO should this be a string? an enum? an int?
	Filter        *TaskCollection `xorm:"json null unique(weight)"`            // Optional reference to a filter. Property must be set to [UrgencyMatchesFilter]. // TODO add security around selecting filter IDs and/or using them. Can be created with access and used after access is removed. Should deleting or changing access to a saved filter alter a user's urgency settings?
	Weight        float64         `xorm:"double not null"`
}

func (*UrgencyWeight) TableName() string {
	return "urgency_weights"
}

// GetUrgencyWeights returns this user's urgency weights.
func GetUrgencyWeights(s *xorm.Session, savedFilterID int64) ([]*UrgencyWeight, error) {
	var urgencyWeights []*UrgencyWeight
	if err := s.Where(builder.Eq{"saved_filter_id": savedFilterID}).Find(&urgencyWeights); err != nil {
		return nil, err
	}
	for _, weight := range urgencyWeights {
		var property UrgencyProperty
		if err := property.UnmarshalText([]byte(weight.Property)); err != nil {
			return nil, errors.Wrap(err, "found invalid property, which should only happen if the API was downgraded")
		}
	}
	return urgencyWeights, nil
}

type urgencyUniqueKey struct {
	Property string      `json:"property"`
	Filter   basicFilter `json:"filter,omitempty"`
}

// TODO this is duplicated
type basicFilter struct {
	Query        string `json:"query"`
	IncludeNulls bool   `json:"include_nulls"`
}

// SetUrgencyWeights validates allWeights, then replaces this saved filter's urgency weights with allWeights.
// allWeights should skip the SavedFilterID field, as those are overridden.
func SetUrgencyWeights(s *xorm.Session, savedFilterID int64, allWeights []UrgencyWeight) (returnedErr error) {
	properties := make(map[urgencyUniqueKey]struct{})
	var newWeights []*UrgencyWeight
	for _, weight := range allWeights {
		weight.SavedFilterID = savedFilterID
		uniqueKey := urgencyUniqueKey{
			Property: weight.Property,
		}
		var property UrgencyProperty
		if err := property.UnmarshalText([]byte(weight.Property)); err != nil {
			return err
		}
		if property == UrgencyMatchesFilter {
			if weight.Filter == nil {
				return errors.New("filter must be set for matches_filter weight")
			}
			uniqueKey.Filter = basicFilter{
				Query:        weight.Filter.Filter,
				IncludeNulls: weight.Filter.FilterIncludeNulls,
			}
			if weight.Filter.Filter == "" {
				return errors.New("filter query must be set")
			}
		}

		if _, exists := properties[uniqueKey]; exists {
			return fmt.Errorf("duplicate weight: %q", weight.Property)
		}
		properties[uniqueKey] = struct{}{}
		newWeights = append(newWeights, &weight)
	}
	return db.DoTransaction(s, func() error {
		if _, err := s.Where(builder.Eq{"saved_filter_id": savedFilterID}).Delete(&UrgencyWeight{}); err != nil {
			sql, _ := s.LastSQL()
			return errors.Wrapf(err, "failed to mark existing weights for replacement: %s", sql)
		}
		if len(newWeights) > 0 {
			if _, err := s.InsertMulti(newWeights); err != nil {
				return errors.Wrap(err, "failed to set weights")
			}
		}
		return nil
	})
}
