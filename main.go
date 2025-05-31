package main

// Oxidized-Netbox API integration wrapper, handles credential sets for Oxidized backups on behalf of Netbox
// v0.50.0

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strings"
)

// define credential set struct
type CredSet struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

// oxidized output fields
type DeviceOut struct {
    Name     string `json:"name"`
    IP       string `json:"ip"`
    Model    string `json:"model"`
    Group    string `json:"group"`
    Username string `json:"username"`
    Password string `json:"password"`
}

var credSets map[string]CredSet

// reads credential sets from disk file
func loadCreds(credentialFilePath string) {
    // attempt to read from file
    credentialFileData, err := os.ReadFile(credentialFilePath)
    if err != nil {
        log.Fatalf("cannot read credentials file %s: %v", credentialFilePath, err)
    }
    // unmarshal json data into credSets struct
    if err := json.Unmarshal(credentialFileData, &credSets); err != nil {
        log.Fatalf("invalid JSON in credentials file: %v", err)
    }
    // if "default" doesn't exist in credSets struct, log warn
    if _, defaultCreds := credSets["default"]; !defaultCreds {
        log.Println("[WARN] no 'default' credential set defined – devices with unknown sets will error")
    }
    log.Printf("[INFO] loaded %d credential sets", len(credSets))
}

// http GET against URL and return raw json bytes
func httpGETRequest(url, apiToken string) ([]byte, error) {
    // formulate new http get request to url & setting token in header
    request, _ := http.NewRequest("GET", url, nil)
    request.Header.Set("Authorization", "Token " + apiToken)
    
    // open http client with request
    response, err := http.DefaultClient.Do(request)
    if err != nil {
        return nil, err
    }
    defer response.Body.Close()

    return io.ReadAll(response.Body)
}

func devicesHandler(writer http.ResponseWriter, request *http.Request) {
    nbURL := os.Getenv("NETBOX_URL")
    nbToken := os.Getenv("NETBOX_TOKEN")
    if nbURL == "" || nbToken == "" {
        http.Error(writer, "NETBOX_URL or NETBOX_TOKEN missing", 500)  // 500: Internal Server Error
        return
    }

    // perform http get and store data into nbJSON & receive raw bytes
    nbJSON, err := httpGETRequest(nbURL, nbToken)
    if err != nil {
        http.Error(writer, "failed talking to NetBox: " + err.Error(), 502) // 502: Bad Gateway
        return
    }

    // create wrapper nb struct that contains nothing but an inner Results struct 
    var nb struct {
        Results []struct {

            Name  string `json:"name"`
            PrimaryIP4 *struct { Address string `json:"address"` } `json:"primary_ip4"`
            Platform *struct { Slug string `json:"slug"` } `json:"platform"`
            Site *struct { Slug string `json:"slug"` } `json:"site"`
            CustomFields map[string]interface{} `json:"custom_fields"`

        } `json:"results"`
    }
    // unmarshal json data into nb structs
    if err := json.Unmarshal(nbJSON, &nb); err != nil {
        http.Error(writer, "invalid NetBox response", 500) // 500: Internal Server Error
        return
    }

    // builds output variable using DeviceOut struct 
    var output []DeviceOut

    // loop over nb.Results content to populate output struct
    // sanitize IP, search for creds, and http write output struct
    for _, device := range nb.Results {
        if device.PrimaryIP4 == nil || device.PrimaryIP4.Address == "" {
            continue // skip devices without IPv4
        }
        sanitizedIP := strings.Split(device.PrimaryIP4.Address, "/")[0]

        // look for nb_cf named "credential_set"
        setName, _ := device.CustomFields["credential_set"].(string)

        // look for setName in credSets, return default if err
        cred, err := credSets[setName]
        if !err {
            cred = credSets["default"]
        }

        output = append(output, DeviceOut{
            Name:     device.Name,
            IP:       sanitizedIP,
            Model:    safeSlug(device.Platform),
            Group:    safeSlug(device.Site),
            Username: cred.Username,
            Password: cred.Password,
        })
    }


    // set HTTP writer content type to json
    writer.Header().Set("Content-Type", "application/json")
    // encode output out http interface
    json.NewEncoder(writer).Encode(map[string]interface{}{"results": output})
}

// validate that slugs are safe, else return unknown slug
func safeSlug(v interface{}) string {
    switch t := v.(type) {
    case *struct{ Slug string `json:"slug"` }:
        if t != nil {
            return t.Slug
        }
    }
    return "unknown"
}

func main() {
    loadCreds(getEnv("CREDENTIALS_FILE", "/etc/oxidized/cred-sets.json"))
    http.HandleFunc("/devices", devicesHandler)
    addr := getEnv("LISTEN", "0.0.0.0:8081")
    log.Println("[INFO] cred-wrapper listening on", addr)
    log.Fatal(http.ListenAndServe(addr, nil))
}

// based on key & credentialfile path, return 
func getEnv(key, def string) string {
    if err := os.Getenv(key); err != "" {
        return err
    }
    return def
}
