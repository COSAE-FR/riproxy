package configuration

import (
	"net"
	"net/http"
)

const defaultProxyPort = 3128
const DefaultTlsPort = 3129
const DefaultBindPort = 80

// Known HTTP methods
// Associated boolean is "default for proxy service"
var httpMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodPost:    true,
	http.MethodPut:     true,
	http.MethodPatch:   true,
	http.MethodDelete:  true,
	http.MethodConnect: true,
	http.MethodOptions: true,
	http.MethodTrace:   false,
	// WebDAV methods
	"COPY":      true, // copy a resource from one URI to another
	"LOCK":      true, // put a lock on a resource
	"MKCOL":     true, // create collections (a.k.a. a directory)
	"MOVE":      true, // move a resource from one URI to another
	"PROPFIND":  true, // retrieve properties from a web resource
	"PROPPATCH": true, // change and delete multiple properties on a resource in a single atomic act
	"UNLOCK":    true, // remove a lock from a resource
	// WebDAV ACL methods
	"ACL": true, // modify the access control list of a resource
	// WebDAV versioning
	"REPORT":           true, // obtain information about a resource
	"VERSION-CONTROL":  true, // create a version-controlled resource
	"CHECKOUT":         true, // allow modifications to the content and dead properties of a checked-in version-controlled resource
	"CHECKIN":          true, // produce a new version whose content and dead properties are copied from the checked-out resource
	"UNCHECKOUT":       true, // cancel the CHECKOUT and restore the pre-CHECKOUT state of the version-controlled resource
	"MKWORKSPACE":      true, // create a new workspace resource
	"UPDATE":           true, // modify the content and dead properties of a checked-in version-controlled resource
	"LABEL":            true, // modify the labels that select a version
	"MERGE":            true, // perform the logical merge of a specified version into a specified version-controlled resource
	"BASELINE-CONTROL": true, // place a collection under baseline control
	"MKACTIVITY":       true, // create a new activity resource
	"SEARCH":           true, // initiate a server-side search
	// WebDAV collection ordering
	"ORDERPATCH": true, // change the ordering semantics of a collection
	// CalDAV methods
	"MKCALENDAR": true, // create a new calendar collection resource
}

// Details about the interface being configured
type interfaceInfo struct {
	Name           string
	Ip             *net.IPNet
	InterfaceProxy string
}

func appendNetwork(slice []net.IPNet, network net.IPNet) []net.IPNet {
	// Normalize to network
	_, candidateNetwork, err := net.ParseCIDR(network.String())
	if err != nil {
		candidateNetwork = &network
	}
	for _, candidate := range slice {
		if candidateNetwork.String() == candidate.String() {
			return slice
		}
	}
	return append(slice, *candidateNetwork)
}
