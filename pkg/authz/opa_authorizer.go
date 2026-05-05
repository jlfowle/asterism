package authz

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"
)

type OPAAuthorizer struct{
    endpoint string
    client   *http.Client
    requiredGroup string
}

// NewOPAAuthorizer constructs an OPAAuthorizer. If the provided URL does not
// contain a v1 path, the typical OPA data path for `asterism/authz/allow` is
// appended.
func NewOPAAuthorizer(rawURL string, requiredGroup string) Authorizer {
    endpoint := strings.TrimSpace(rawURL)
    if !strings.Contains(endpoint, "/v1/") {
        endpoint = strings.TrimRight(endpoint, "/") + "/v1/data/asterism/authz/allow"
    }

    return &OPAAuthorizer{
        endpoint: endpoint,
        client: &http.Client{Timeout: 2 * time.Second},
        requiredGroup: strings.TrimSpace(requiredGroup),
    }
}

// IsAllowed calls OPA with the input and expects a JSON response. It understands
// either `{"result": true}` or `{"result": {"allow": true}}` shapes.
func (o *OPAAuthorizer) IsAllowed(ctx context.Context, principal string, groups []string, resource string, verb string) (bool, string, error) {
    input := map[string]any{"input": map[string]any{"principal": principal, "groups": groups, "resource": resource, "verb": verb}}

    reqBody, err := json.Marshal(input)
    if err != nil {
        return false, "internal error", err
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.endpoint, strings.NewReader(string(reqBody)))
    if err != nil {
        return false, "internal error", err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := o.client.Do(req)
    if err != nil {
        return false, "authorization backend error", err
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 400 {
        return false, fmt.Sprintf("authorization backend returned %d", resp.StatusCode), nil
    }

    var buf map[string]any
    if err := json.NewDecoder(resp.Body).Decode(&buf); err != nil {
        return false, "invalid authorization response", err
    }

    // inspect buf["result"]
    if res, ok := buf["result"]; ok {
        switch v := res.(type) {
        case bool:
            if v { return true, "", nil }
            return false, "denied", nil
        case map[string]any:
            if allow, ok := v["allow"].(bool); ok {
                if allow { return true, "", nil }
                return false, "denied", nil
            }
        }
    }

    // fallback: if OPA did not allow and we have a requiredGroup, enforce local group check
    if o.requiredGroup != "" {
        for _, g := range groups {
            if g == o.requiredGroup {
                return true, "", nil
            }
        }
        return false, "missing required group", nil
    }

    return false, "denied", nil
}
