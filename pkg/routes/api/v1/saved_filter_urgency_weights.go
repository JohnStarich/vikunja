package v1

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	user2 "code.vikunja.io/api/pkg/user"
	"github.com/labstack/echo/v5"
)

type SavedFilterUrgencyWeights struct {
	// TODO merge this into a saved filter? Is it useful to keep this split out to manipulate a single weight at a time?
	UrgencyWeights []UrgencyWeight `json:"urgency_weights"`
}

type UrgencyWeight struct {
	Property string       `json:"property"` // TODO should this be the enum type with UnmarshalText()?
	Weight   float64      `json:"weight"`
	Filter   *BasicFilter `json:"filter,omitempty"`
}

type BasicFilter struct {
	Query        string `json:"query"`
	IncludeNulls bool   `json:"include_nulls"`
}

func getSavedFilterID(c *echo.Context) (int64, error) {
	idStr := c.Param("filter")
	const (
		decimalBase = 10
		int64Size   = 64
	)
	id, err := strconv.ParseInt(idStr, decimalBase, int64Size)
	if err != nil {
		return 0, models.ErrInvalidModel{Err: fmt.Errorf("saved_filter_id must be an integer, got %q: %w", idStr, err)}
	}
	return id, nil
}

// GetSavedFilterUrgencyWeights returns the currently set saved filter urgency weights
// @Summary Return saved filter urgency weights
// @Description Returns the saved filter's urgency weights.
// @tags filter
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Success 200 {object} SavedFilterUrgencyWeights
// @Failure 400 {object} web.HTTPError "Something's invalid."
// @Failure 500 {object} models.Message "Internal server error."
// @Router /user/settings/avatar [get]
func GetSavedFilterUrgencyWeights(c *echo.Context) error {
	s := db.NewSession()
	defer s.Close()

	id, err := getSavedFilterID(c)
	if err != nil {
		return err
	}
	weights, err := models.GetUrgencyWeights(s, id)
	if err != nil {
		return err
	}
	urgencyWeights := make([]UrgencyWeight, 0, len(weights))
	for _, weight := range weights {
		var filter *BasicFilter
		if weight.Filter != nil {
			filter = &BasicFilter{
				Query:        weight.Filter.Filter,
				IncludeNulls: weight.Filter.FilterIncludeNulls,
			}
		}
		urgencyWeights = append(urgencyWeights, UrgencyWeight{
			Property: weight.Property,
			Weight:   weight.Weight,
			Filter:   filter,
		})
	}
	return c.JSON(http.StatusOK, SavedFilterUrgencyWeights{
		UrgencyWeights: urgencyWeights,
	})
}

// UpdateSavedFilterUrgencyWeights is the handler to change a saved filter's urgency weights
// @Summary Change a saved filter's urgency weights
// @tags filter
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param urgency_weights body UrgencyWeights true "The updated saved filter urgency weights"
// @Success 200 {object} models.Message
// @Failure 400 {object} web.HTTPError "Something's invalid."
// @Failure 500 {object} models.Message "Internal server error."
// @Router /user/settings/urgency_weights [post]
func UpdateSavedFilterUrgencyWeights(c *echo.Context) error {
	id, err := getSavedFilterID(c)
	if err != nil {
		return err
	}

	var urgencyWeights SavedFilterUrgencyWeights
	if err := c.Bind(&urgencyWeights); err != nil {
		var he *echo.HTTPError
		if errors.As(err, &he) {
			return models.ErrInvalidModel{Message: fmt.Sprintf("%v", he.Message), Err: err}
		}
		return models.ErrInvalidModel{Err: err}
	}

	if err := c.Validate(urgencyWeights); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error()).Wrap(err)
	}

	u, err := user2.GetCurrentUser(c)
	if err != nil {
		return err
	}

	s := db.NewSession()
	defer s.Close()

	var weights []models.UrgencyWeight
	// TODO add validation for filter property
	for _, weight := range urgencyWeights.UrgencyWeights {
		var filter *models.TaskCollection
		if weight.Filter != nil {
			filter = &models.TaskCollection{
				Filter:             weight.Filter.Query,
				FilterTimezone:     u.Timezone, // TODO replace with saved filter's time zone?
				FilterIncludeNulls: weight.Filter.IncludeNulls,
			}
			if err := filter.ValidateFilterString(); err != nil {
				return models.ErrInvalidModel{Err: err}
			}
		}
		if weight.Weight < 1 {
			return models.ErrInvalidModel{Err: fmt.Errorf("property %q weight was %.2f, must be at least 1", weight.Property, weight.Weight)}
		}
		weights = append(weights, models.UrgencyWeight{
			Property: weight.Property,
			Weight:   weight.Weight,
			Filter:   filter,
		})
	}
	if err := models.SetUrgencyWeights(s, id, weights); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error()).Wrap(err)
	}

	return c.JSON(http.StatusOK, &models.Message{Message: "The urgency weights were updated successfully."})
}
