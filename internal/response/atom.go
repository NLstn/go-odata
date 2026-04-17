package response

import (
"encoding/xml"
"fmt"
"io"
"net/http"
"reflect"
"strconv"
"strings"
"time"
)

const (
atomNamespace   = "http://www.w3.org/2005/Atom"
odataDataNS     = "http://docs.oasis-open.org/odata/ns/data"
odataMetaNS     = "http://docs.oasis-open.org/odata/ns/metadata"
odataSchemeNS   = "http://docs.oasis-open.org/odata/ns/scheme"
atomContentType = "application/atom+xml;charset=utf-8"
)

// IsAtomFormat returns true if the request asks for Atom/XML format via
// $format=atom, $format=application/atom+xml, or Accept: application/atom+xml.
func IsAtomFormat(r *http.Request) bool {
format := getFormatParameter(r.URL.RawQuery)
if format != "" {
parts := strings.Split(format, ";")
baseFormat := strings.TrimSpace(parts[0])
return baseFormat == "atom" || baseFormat == "application/atom+xml"
}
accept := r.Header.Get("Accept")
if accept == "" {
return false
}
for _, part := range strings.Split(accept, ",") {
part = strings.TrimSpace(part)
subparts := strings.Split(part, ";")
if strings.TrimSpace(subparts[0]) == "application/atom+xml" {
return true
}
}
return false
}

// atomWriter wraps xml.Encoder to accumulate errors from EncodeToken calls.
// This avoids propagating individual token errors through every helper function
// while still detecting any write failure at Flush time.
type atomWriter struct {
enc *xml.Encoder
err error
}

func newAtomWriter(w io.Writer) *atomWriter {
return &atomWriter{enc: xml.NewEncoder(w)}
}

func (a *atomWriter) token(t xml.Token) {
if a.err != nil {
return
}
a.err = a.enc.EncodeToken(t)
}

func (a *atomWriter) start(localName string, attrs ...xml.Attr) {
a.token(xml.StartElement{Name: xml.Name{Local: localName}, Attr: attrs})
}

func (a *atomWriter) end(localName string) {
a.token(xml.EndElement{Name: xml.Name{Local: localName}})
}

func (a *atomWriter) text(localName, text string) {
a.start(localName)
a.token(xml.CharData(text))
a.end(localName)
}

func (a *atomWriter) empty(localName string) {
a.start(localName)
a.end(localName)
}

func (a *atomWriter) link(rel, href string) {
a.token(xml.StartElement{
Name: xml.Name{Local: "link"},
Attr: []xml.Attr{
{Name: xml.Name{Local: "rel"}, Value: rel},
{Name: xml.Name{Local: "href"}, Value: href},
},
})
a.end("link")
}

func (a *atomWriter) flush() error {
if a.err != nil {
return a.err
}
return a.enc.Flush()
}

// WriteAtomCollection writes an OData collection as an Atom feed.
// keyProps are used to build per-entry self-link IDs when @odata.id is not already present.
func WriteAtomCollection(w http.ResponseWriter, r *http.Request, entitySetName string, data interface{}, count *int64, nextLink, deltaLink *string, keyProps []PropertyMetadata) error {
baseURL := buildBaseURL(r)
contextURL := baseURL + "/$metadata#" + entitySetName
feedURL := baseURL + "/" + entitySetName

SetODataVersionHeaderFromRequest(w, r)
w.Header().Set("Content-Type", atomContentType)

if r.Method == http.MethodHead {
w.WriteHeader(http.StatusOK)
return nil
}

w.WriteHeader(http.StatusOK)
if _, err := io.WriteString(w, "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"); err != nil {
return err
}

aw := newAtomWriter(w)
aw.start("feed",
xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: atomNamespace},
xml.Attr{Name: xml.Name{Local: "xmlns:d"}, Value: odataDataNS},
xml.Attr{Name: xml.Name{Local: "xmlns:m"}, Value: odataMetaNS},
)

aw.text("m:context", contextURL)
aw.text("id", feedURL)
aw.empty("title")
aw.text("updated", time.Now().UTC().Format(time.RFC3339))
aw.link("self", feedURL)

if count != nil {
aw.text("m:count", strconv.FormatInt(*count, 10))
}

if data != nil {
dataValue := reflect.ValueOf(data)
if dataValue.Kind() == reflect.Ptr {
dataValue = dataValue.Elem()
}
if dataValue.Kind() == reflect.Slice {
now := time.Now().UTC().Format(time.RFC3339)
for i := 0; i < dataValue.Len(); i++ {
item := dataValue.Index(i).Interface()
entityID := extractAtomEntityID(item, baseURL, entitySetName, keyProps)
writeAtomEntry(aw, entitySetName, entityID, item, now)
}
}
}

if nextLink != nil && *nextLink != "" {
aw.link("next", *nextLink)
}

aw.end("feed")
return aw.flush()
}

// WriteAtomEntity writes a single OData entity as an Atom entry.
// entityID should be the full URL of the entity (e.g. "http://host/service/Products(1)").
func WriteAtomEntity(w http.ResponseWriter, r *http.Request, entitySetName string, entityID string, data interface{}, etagValue string, status int) error {
baseURL := buildBaseURL(r)
contextURL := baseURL + "/$metadata#" + entitySetName + "/$entity"
now := time.Now().UTC().Format(time.RFC3339)

SetODataVersionHeaderFromRequest(w, r)
if etagValue != "" {
w.Header().Set("ETag", etagValue)
}
w.Header().Set("Content-Type", atomContentType)

if r.Method == http.MethodHead {
if status == 0 {
status = http.StatusOK
}
w.WriteHeader(status)
return nil
}

if status == 0 {
status = http.StatusOK
}
w.WriteHeader(status)
if _, err := io.WriteString(w, "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"); err != nil {
return err
}

aw := newAtomWriter(w)
aw.start("entry",
xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: atomNamespace},
xml.Attr{Name: xml.Name{Local: "xmlns:d"}, Value: odataDataNS},
xml.Attr{Name: xml.Name{Local: "xmlns:m"}, Value: odataMetaNS},
)

aw.text("m:context", contextURL)
aw.text("id", entityID)
aw.empty("title")
aw.text("updated", now)
writeAtomAuthor(aw)
writeAtomCategory(aw, entitySetName)
aw.link("edit", entityID)
writeAtomProperties(aw, data)

aw.end("entry")
return aw.flush()
}

// writeAtomEntry writes a single <entry> element within a feed.
func writeAtomEntry(aw *atomWriter, entitySetName, entityID string, data interface{}, now string) {
aw.start("entry")
aw.text("id", entityID)
aw.empty("title")
aw.text("updated", now)
writeAtomAuthor(aw)
writeAtomCategory(aw, entitySetName)
if entityID != "" {
aw.link("edit", entityID)
}
writeAtomProperties(aw, data)
aw.end("entry")
}

func writeAtomAuthor(aw *atomWriter) {
aw.start("author")
aw.empty("name")
aw.end("author")
}

func writeAtomCategory(aw *atomWriter, entitySetName string) {
aw.token(xml.StartElement{
Name: xml.Name{Local: "category"},
Attr: []xml.Attr{
{Name: xml.Name{Local: "term"}, Value: entitySetName},
{Name: xml.Name{Local: "scheme"}, Value: odataSchemeNS},
},
})
aw.end("category")
}

func writeAtomProperties(aw *atomWriter, data interface{}) {
aw.start("content", xml.Attr{Name: xml.Name{Local: "type"}, Value: "application/xml"})
aw.start("m:properties")

for _, prop := range extractAtomDataProperties(data) {
propName := "d:" + prop.name
if prop.value == nil {
aw.start(propName, xml.Attr{Name: xml.Name{Local: "m:null"}, Value: "true"})
} else {
aw.start(propName)
aw.token(xml.CharData(fmt.Sprintf("%v", prop.value)))
}
aw.end(propName)
}

aw.end("m:properties")
aw.end("content")
}

type atomDataProp struct {
name  string
value interface{}
}

// extractAtomDataProperties extracts data properties from an entity,
// skipping OData control information (keys starting with "@" or "__temp_").
func extractAtomDataProperties(data interface{}) []atomDataProp {
if data == nil {
return nil
}
switch v := data.(type) {
case *OrderedMap:
props := make([]atomDataProp, 0, len(v.keys))
for _, key := range v.keys {
if isAtomControlKey(key) {
continue
}
props = append(props, atomDataProp{name: key, value: v.values[key]})
}
return props
case map[string]interface{}:
props := make([]atomDataProp, 0, len(v))
for key, value := range v {
if isAtomControlKey(key) {
continue
}
props = append(props, atomDataProp{name: key, value: value})
}
return props
default:
return extractAtomStructProperties(data)
}
}

func extractAtomStructProperties(data interface{}) []atomDataProp {
val := reflect.ValueOf(data)
if val.Kind() == reflect.Ptr {
val = val.Elem()
}
if val.Kind() != reflect.Struct {
return nil
}
t := val.Type()
props := make([]atomDataProp, 0, t.NumField())
for i := 0; i < t.NumField(); i++ {
field := t.Field(i)
if !field.IsExported() {
continue
}
name := field.Name
if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
if parts := strings.SplitN(tag, ",", 2); parts[0] != "" {
name = parts[0]
}
}
props = append(props, atomDataProp{name: name, value: val.Field(i).Interface()})
}
return props
}

// isAtomControlKey returns true for OData control information keys that
// should not appear as data properties in Atom responses.
func isAtomControlKey(key string) bool {
return strings.HasPrefix(key, "@") || strings.HasPrefix(key, "__temp_")
}

// extractAtomEntityID builds the full entity ID URL for an Atom entry.
// It uses @odata.id from *OrderedMap/map if present, or falls back to key property extraction.
func extractAtomEntityID(data interface{}, baseURL, entitySetName string, keyProps []PropertyMetadata) string {
// Check if @odata.id is already present in the data
switch v := data.(type) {
case *OrderedMap:
if id, ok := v.values["@odata.id"]; ok {
if idStr, ok2 := id.(string); ok2 && idStr != "" {
return idStr
}
}
case map[string]interface{}:
if id, ok := v["@odata.id"]; ok {
if idStr, ok2 := id.(string); ok2 && idStr != "" {
return idStr
}
}
}

// Fall back to extracting from key properties via reflection
if len(keyProps) == 0 {
return ""
}

keyValues := make(map[string]interface{})
val := reflect.ValueOf(data)
if val.Kind() == reflect.Ptr {
val = val.Elem()
}
if val.Kind() == reflect.Struct {
t := val.Type()
for i := 0; i < t.NumField(); i++ {
field := t.Field(i)
for _, kp := range keyProps {
if field.Name == kp.Name {
keyValues[kp.JsonName] = val.Field(i).Interface()
}
}
}
}

if len(keyValues) == 0 {
return ""
}
return baseURL + "/" + BuildEntityID(entitySetName, keyValues)
}
