package main

import (
	"encoding/json"
	"net/http"

	"github.com/nlstn/go-odata"
	"gorm.io/gorm"
)

// registerReseedAction registers an unbound action to reseed the database
// This is useful for testing to reset the database to a known state
func registerReseedAction(service *odata.Service, db *gorm.DB, dbType string) {
	if err := service.RegisterAction(odata.ActionDefinition{
		Name:       "Reseed",
		IsBound:    false,
		Parameters: []odata.ParameterDefinition{},
		ReturnType: nil,
		Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
			// Reseed the database
			if err := seedDatabase(db, dbType); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}

			// Return success response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			response := map[string]interface{}{
				"status":  "success",
				"message": "Database reseeded with default data",
			}

			return json.NewEncoder(w).Encode(response)
		},
	}); err != nil {
		panic("Failed to register Reseed action: " + err.Error())
	}
}
