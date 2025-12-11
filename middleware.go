package main

import (
	"context"
	"net/http"
)

func authMiddleware(requiredType string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		
		user, err := db.GetUserByID(cookie.Value)
		if err != nil || string(user.Type) != requiredType {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
