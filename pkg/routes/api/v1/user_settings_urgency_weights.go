package v1

import (
	"errors"
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	user2 "code.vikunja.io/api/pkg/user"
	"github.com/labstack/echo/v5"
)

type UserUrgencyWeights struct {
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

// GetUserUrgencyWeightsSettings returns the currently set user avatar
// @Summary Return user urgency weights setting
// @Description Returns the current user's urgency weights setting.
// @tags user
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Success 200 {object} UserUrgencyWeights
// @Failure 400 {object} web.HTTPError "Something's invalid."
// @Failure 500 {object} models.Message "Internal server error."
// @Router /user/settings/avatar [get]
func GetUserUrgencyWeightsSettings(c *echo.Context) error {
	u, err := user2.GetCurrentUser(c)
	if err != nil {
		return err
	}

	s := db.NewSession()
	defer s.Close()

	weights, err := models.GetUrgencyWeights(s, u.ID)
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
	return c.JSON(http.StatusOK, UserUrgencyWeights{
		UrgencyWeights: urgencyWeights,
	})
}

// UpdateUserUrgencyWeightsSettings is the handler to change general user settings
// @Summary Change user urgency weight settings of the current user.
// @tags user
// @Accept json
// @Produce json
// @Security JWTKeyAuth
// @Param urgency_weights body UrgencyWeights true "The updated user urgency weights"
// @Success 200 {object} models.Message
// @Failure 400 {object} web.HTTPError "Something's invalid."
// @Failure 500 {object} models.Message "Internal server error."
// @Router /user/settings/urgency_weights [post]
func UpdateUserUrgencyWeightsSettings(c *echo.Context) error {
	var userUrgencyWeights UserUrgencyWeights
	if err := c.Bind(&userUrgencyWeights); err != nil {
		var he *echo.HTTPError
		if errors.As(err, &he) {
			return models.ErrInvalidModel{Message: fmt.Sprintf("%v", he.Message), Err: err}
		}
		return models.ErrInvalidModel{Err: err}
	}

	if err := c.Validate(userUrgencyWeights); err != nil {
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
	for _, weight := range userUrgencyWeights.UrgencyWeights {
		var filter *models.TaskCollection
		if weight.Filter != nil {
			filter = &models.TaskCollection{
				Filter:             weight.Filter.Query,
				FilterTimezone:     u.Timezone,
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
	if err := models.SetUrgencyWeights(s, u.ID, weights); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error()).Wrap(err)
	}

	return c.JSON(http.StatusOK, &models.Message{Message: "The urgency weights were updated successfully."})
}
