package controllers

import (
	"net/http"
)

func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "../../web/favicon/favicon.ico")
}
