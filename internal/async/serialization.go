package async

import (
	"encoding/json"
	"net/http"
)

func serializeHeaders(header http.Header) ([]byte, error) {
	if len(header) == 0 {
		return nil, nil
	}
	return json.Marshal(header)
}

func deserializeHeaders(data []byte) (http.Header, error) {
	if len(data) == 0 {
		return http.Header{}, nil
	}
	var raw map[string][]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	hdr := make(http.Header, len(raw))
	for k, values := range raw {
		hdr[k] = append([]string(nil), values...)
	}
	return hdr, nil
}

func storedResponseToRecord(resp *StoredResponse) (*int, []byte, []byte, error) {
	if resp == nil {
		return nil, nil, nil, nil
	}
	headers, err := serializeHeaders(resp.Header)
	if err != nil {
		return nil, nil, nil, err
	}
	status := resp.StatusCode
	var body []byte
	if len(resp.Body) > 0 {
		body = append([]byte(nil), resp.Body...)
	}
	return &status, headers, body, nil
}

func storedResponseFromRecord(status *int, headersData, body []byte) (*StoredResponse, error) {
	if status == nil && len(headersData) == 0 && len(body) == 0 {
		return nil, nil
	}
	header, err := deserializeHeaders(headersData)
	if err != nil {
		return nil, err
	}
	resp := &StoredResponse{
		Header: header,
	}
	if status != nil {
		resp.StatusCode = *status
	}
	if len(body) > 0 {
		resp.Body = append([]byte(nil), body...)
	}
	return resp, nil
}
