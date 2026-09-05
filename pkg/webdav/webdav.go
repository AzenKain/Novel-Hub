package webdav

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// WebDAVNode represents a collection (folder) or a resource (file) in WebDAV.
type WebDAVNode struct {
	Href        string
	DisplayName string
	IsDir       bool
	Size        int64
	ContentType string
	ModTime     time.Time
	ETag        string
}

// XML Structs conforming to RFC 4918 (WebDAV)

type responseXML struct {
	XMLName  xml.Name    `xml:"D:response"`
	Href     string      `xml:"D:href"`
	Propstat propstatXML `xml:"D:propstat"`
}

type propstatXML struct {
	Prop   propXML `xml:"D:prop"`
	Status string  `xml:"D:status"`
}

type propXML struct {
	DisplayName   string           `xml:"D:displayname,omitempty"`
	ResourceType  *resourceTypeXML `xml:"D:resourcetype"`
	ContentLength *int64           `xml:"D:getcontentlength,omitempty"`
	ContentType   string           `xml:"D:getcontenttype,omitempty"`
	LastModified  string           `xml:"D:getlastmodified,omitempty"`
	ETag          string           `xml:"D:getetag,omitempty"`
}

type resourceTypeXML struct {
	Collection *struct{} `xml:"D:collection,omitempty"`
}

type multistatusXML struct {
	XMLName   xml.Name      `xml:"D:multistatus"`
	XmlnsD    string        `xml:"xmlns:D,attr"`
	Responses []responseXML `xml:"D:response"`
}

// BuildMultiStatusXML serializes a list of WebDAV nodes into a compliant RFC 4918 Multi-Status XML payload.
func BuildMultiStatusXML(nodes []WebDAVNode) ([]byte, error) {
	responses := make([]responseXML, 0, len(nodes))

	for _, node := range nodes {
		href := node.Href
		if !strings.HasPrefix(href, "/") {
			href = "/" + href
		}
		if node.IsDir && !strings.HasSuffix(href, "/") {
			href = href + "/"
		}

		displayName := node.DisplayName
		if displayName == "" {
			displayName = path.Base(strings.TrimSuffix(href, "/"))
			if displayName == "" || displayName == "." {
				displayName = "webdav"
			}
		}

		p := propXML{
			DisplayName: displayName,
		}

		if node.IsDir {
			p.ResourceType = &resourceTypeXML{
				Collection: &struct{}{},
			}
			p.ContentType = "httpd/unix-directory"
		} else {
			p.ResourceType = &resourceTypeXML{}
			length := node.Size
			p.ContentLength = &length
			if node.ContentType != "" {
				p.ContentType = node.ContentType
			} else {
				p.ContentType = "application/octet-stream"
			}
		}

		if !node.ModTime.IsZero() {
			p.LastModified = node.ModTime.UTC().Format(http.TimeFormat)
		} else {
			p.LastModified = time.Now().UTC().Format(http.TimeFormat)
		}

		if node.ETag != "" {
			p.ETag = node.ETag
		}

		responses = append(responses, responseXML{
			Href: href,
			Propstat: propstatXML{
				Prop:   p,
				Status: "HTTP/1.1 200 OK",
			},
		})
	}

	ms := multistatusXML{
		XmlnsD:    "DAV:",
		Responses: responses,
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(ms); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ParseDepth parses the WebDAV Depth header (defaults to 1 if unspecified or invalid).
func ParseDepth(depthHeader string) int {
	clean := strings.ToLower(strings.TrimSpace(depthHeader))
	switch clean {
	case "0":
		return 0
	case "1":
		return 1
	case "infinity":
		return 1
	default:
		return 1
	}
}

// SanitizeWebDAVPath cleans, unescapes, and normalizes a WebDAV relative URL path.
func SanitizeWebDAVPath(rawPath string) string {
	if unescaped, err := url.PathUnescape(rawPath); err == nil {
		rawPath = unescaped
	}
	rawPath = strings.TrimPrefix(rawPath, "/api/webdav")
	rawPath = strings.TrimPrefix(rawPath, "api/webdav")
	rawPath = strings.TrimPrefix(rawPath, "/webdav")
	rawPath = strings.TrimPrefix(rawPath, "webdav")
	cleaned := path.Clean("/" + rawPath)
	if cleaned == "." || cleaned == "" {
		return "/"
	}
	return cleaned
}
