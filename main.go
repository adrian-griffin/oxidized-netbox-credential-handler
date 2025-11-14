package main

// Oxidized-Netbox API integration wrapper, handles credential sets for Oxidized backups on behalf of Netbox

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"log"
	"net"
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
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Model     string `json:"model"`
	Group     string `json:"group"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	EnablePwd string `json:"enable_password"`
	SSHPort   string `json:"ssh_port"`
}

// build out CredSet struct into var
var credSets map[string]CredSet

// init custom http client
var httpClient *http.Client

// define inbound api auth token via env var
var allowedAPIKey = getEnv("WRAPPER_TOKEN", "")

func initHTTPClient() {
	caFile := os.Getenv("NETBOX_CA_FILE")
	if caFile == "" {
		// Fallback: behave exactly like before
		httpClient = http.DefaultClient
		log.Println("[INFO] NETBOX_CA_FILE not set – using system root CAs")
		return
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		log.Fatalf("failed to read CA file %s: %v", caFile, err)
	}

	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caCert) {
		log.Fatalf("failed to parse CA certificate(s) in %s", caFile)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: roots,
		},
	}

	httpClient = &http.Client{Transport: tr}
	log.Printf("[INFO] custom HTTP client initialized with CA %s", caFile)
}

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
	request.Header.Set("Authorization", "Token "+apiToken)

	// open http client with request
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return io.ReadAll(response.Body)
}

// best effort at extracting client IP
func getClientIP(r *http.Request) string {
	// X-Forwarded-For may contain a list. take the first entry.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	if rip := r.Header.Get("X-Real-IP"); rip != "" {
		return rip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // fallback as‑is
	}
	return host
}

// handles when query against x:8081/devices
func devicesHandler(writer http.ResponseWriter, request *http.Request) {
	// log every request
	log.Printf("[REQ] %s %s from %s", request.Method, request.URL.Path, getClientIP(request))

	// check api token header for authorization
	clientAPIKey := request.Header.Get("Authorization")
	if clientAPIKey != "Token "+allowedAPIKey {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		log.Printf("[WARN] unauthorized request from %s", getClientIP(request))
		return
	}

	nbURL := os.Getenv("NETBOX_URL")
	nbToken := os.Getenv("NETBOX_TOKEN")
	if nbURL == "" || nbToken == "" {
		http.Error(writer, "NETBOX_URL or NETBOX_TOKEN missing", 500) // 500: Internal Server Error
		return
	}

	// perform http get and store data into nbJSON & receive raw bytes
	nbJSON, err := httpGETRequest(nbURL, nbToken)
	if err != nil {
		http.Error(writer, "failed talking to NetBox: "+err.Error(), 502) // 502: Bad Gateway
		return
	}
	log.Printf("[INFO] Good GET request to %s", nbURL)

	// create wrapper nb struct that contains nothing but an inner Results struct
	var nb struct {
		Results []struct {
			Name       string `json:"name"`
			PrimaryIP4 *struct {
				Address string `json:"address"`
			} `json:"primary_ip4"`
			Platform *struct {
				Slug string `json:"slug"`
			} `json:"platform"`
			Site *struct {
				Slug string `json:"slug"`
			} `json:"site"`
			CustomFields map[string]interface{} `json:"custom_fields"` // (no, i still dont get this)
		} `json:"results"`
	}
	// unmarshal json data into nb structs
	if err := json.Unmarshal(nbJSON, &nb); err != nil {
		// log a snippet of what we actually got from NetBox
		snippet := string(nbJSON)
		if len(snippet) > 800 {
			snippet = snippet[:800] + "...(truncated)"
		}
		log.Printf("[ERROR] failed to unmarshal NetBox JSON: %v; raw response snippet: %q", err, snippet)

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

		enablePassword, _ := device.CustomFields["enable_password"].(string)

		sshPort, _ := device.CustomFields["oxidized_ssh_port"].(string)

		// look for setName in credSets, return default if err
		cred, err := credSets[setName]
		if !err {
			log.Println("[WARN] unable to find credential set specified on NetBox device in credentials.json file")
			cred = credSets["default"]
		}

		output = append(output, DeviceOut{
			Name:      device.Name,
			IP:        sanitizedIP,
			Model:     safeSlug(device.Platform),
			Group:     safeSlug(device.Site),
			Username:  cred.Username,
			Password:  cred.Password,
			EnablePwd: enablePassword,
			SSHPort:   sshPort,
		})
	}
	// set HTTP writer content type to json
	writer.Header().Set("Content-Type", "application/json")
	// encode output out http interface
	json.NewEncoder(writer).Encode(map[string]interface{}{"results": output})

	log.Printf("[INFO] returned %d valid nodes", len(nb.Results))
}

// validate that slugs are !netbox's null,  return blank string at least
func safeSlug(v interface{}) string {
	switch t := v.(type) {
	case *struct {
		Slug string `json:"slug"`
	}:
		if t != nil {
			return t.Slug
		}
	}
	return ""
}

// health check endpoint
func healthPoll(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

func main() {

	const Version = "v0.53.1"

	loadCreds(getEnv("CREDENTIALS_FILE", "./cred-sets.json"))

	initHTTPClient()

	http.HandleFunc("/devices", devicesHandler)
	http.HandleFunc("/healthz", healthPoll)

	addr := getEnv("LISTEN", "0.0.0.0:8081")
	log.Printf("[INFO] cred-wrapper %s listening on %s", Version, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// get env variables and return, otherwise use default passed
func getEnv(key, def string) string {
	if err := os.Getenv(key); err != "" {
		return err
	}
	return def
}
