package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/jinzhu/gorm"

	yamlV2 "gopkg.in/yaml.v2"
)

// MapHandler will return an http.HandlerFunc (which also
// implements http.Handler) that will attempt to map any
// paths (keys in the map) to their corresponding URL (values
// that each key in the map points to, in string format).
// If the path is not provided in the map, then the fallback
// http.Handler will be called instead.
func MapHandler(pathsToUrls map[string]string, fallback http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path, ok := pathsToUrls[r.URL.Path]
		if ok {
			http.Redirect(w, r, path, http.StatusFound)
		} else {
			fallback.ServeHTTP(w, r)
		}
	}
}

// YAMLHandler will parse the provided YAML and then return
// an http.HandlerFunc (which also implements http.Handler)
// that will attempt to map any paths to their corresponding
// URL. If the path is not provided in the YAML, then the
// fallback http.Handler will be called instead.
//
// YAML is expected to be in the format:
//
//     - path: /some-path
//       url: https://www.some-url.com/demo
//
// The only errors that can be returned all related to having
// invalid YAML data.
//
// See MapHandler to create a similar http.HandlerFunc via
// a mapping of paths to urls.
func YAMLHandler(yaml []byte, fallback http.Handler) (http.HandlerFunc, error) {
	parsedYaml, err := parseYAML(yaml)
	if err != nil {
		return nil, err
	}
	pathMap := buildMap(parsedYaml)
	return MapHandler(pathMap, fallback), nil
}

// JSONHandler will parse the provided JSON and then return
// an http.HandlerFunc (which also implements http.Handler)
// that will attempt to map any paths to their corresponding
// URL. If the path is not provided in the JSON, then the
// fallback http.Handler will be called instead.
//
// JSON is expected to be in the format:
//
//		{
//			"/some-path":"https://www.some-url.com/demo"
//		}
//
// The only errors that can be returned all related to having
// invalid JSON data.
//
// See MapHandler to create a similar http.HandlerFunc via
// a mapping of paths to urls.
func JSONHandler(jsonData []byte, fallback http.Handler) (http.HandlerFunc, error) {
	parsedJSON, err := parseJSON(jsonData)
	if err != nil {
		return nil, err
	}
	return MapHandler(parsedJSON, fallback), nil
}

// DBHandler will return an http.HandlerFunc that queries the database for the
// request URL and redirects as necessary
func DBHandler(db *gorm.DB, fallback http.Handler) (http.HandlerFunc, error) {
	type urlmap struct {
		Shortpath string `gorm:"not null;unique_index"`
		URL       string `gorm:"not null"`
	}
	if err := db.AutoMigrate(&urlmap{}).Error; err != nil {
		log.Println("Gorm error: ", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		urlMap := urlmap{
			Shortpath: r.URL.Path,
		}
		var dst urlmap
		err := db.Where(urlMap).First(&dst).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				fallback.ServeHTTP(w, r)
			} else {
				fmt.Fprintf(w, "Unexpected error: %s", err)
			}
			return
		}
		http.Redirect(w, r, dst.URL, http.StatusMovedPermanently)

	}, nil
}
func parseYAML(yaml []byte) (dst []map[string]string, err error) {
	if err = yamlV2.Unmarshal(yaml, &dst); err != nil {
		return nil, err
	}
	return dst, nil
}

func parseJSON(jsonData []byte) (dst map[string]string, err error) {
	if err = json.Unmarshal(jsonData, &dst); err != nil {
		return nil, err
	}
	return dst, nil
}
func buildMap(parsedYaml []map[string]string) map[string]string {
	mergedMap := make(map[string]string)
	for _, entry := range parsedYaml {
		key := entry["path"]
		mergedMap[key] = entry["url"]
	}
	return mergedMap
}
