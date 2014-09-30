package vips

import (
	"fmt"
	"strconv"

	"github.com/racker/perigee"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/utils"
	"github.com/rackspace/gophercloud/pagination"
)

const (
	Up   = true
	Down = false
)

// ListOpts allows the filtering and sorting of paginated collections through
// the API. Filtering is achieved by passing in struct field values that map to
// the floating IP attributes you want to see returned. SortKey allows you to
// sort by a particular network attribute. SortDir sets the direction, and is
// either `asc' or `desc'. Marker and Limit are used for pagination.
type ListOpts struct {
	ID              string
	Name            string
	AdminStateUp    *bool
	Status          string
	TenantID        string
	SubnetID        string
	Address         string
	PortID          string
	Protocol        string
	ProtocolPort    int
	ConnectionLimit int
	Limit           int
	Marker          string
	SortKey         string
	SortDir         string
}

// List returns a Pager which allows you to iterate over a collection of
// routers. It accepts a ListOpts struct, which allows you to filter and sort
// the returned collection for greater efficiency.
//
// Default policy settings return only those routers that are owned by the
// tenant who submits the request, unless an admin user submits the request.
func List(c *gophercloud.ServiceClient, opts ListOpts) pagination.Pager {
	q := make(map[string]string)
	if opts.ID != "" {
		q["id"] = opts.ID
	}
	if opts.Name != "" {
		q["name"] = opts.Name
	}
	if opts.AdminStateUp != nil {
		q["admin_state_up"] = strconv.FormatBool(*opts.AdminStateUp)
	}
	if opts.Status != "" {
		q["status"] = opts.Status
	}
	if opts.TenantID != "" {
		q["tenant_id"] = opts.TenantID
	}
	if opts.SubnetID != "" {
		q["subnet_id"] = opts.SubnetID
	}
	if opts.Address != "" {
		q["address"] = opts.Address
	}
	if opts.PortID != "" {
		q["port_id"] = opts.PortID
	}
	if opts.Protocol != "" {
		q["protocol"] = opts.Protocol
	}
	if opts.ProtocolPort != 0 {
		q["protocol_port"] = strconv.Itoa(opts.ProtocolPort)
	}
	if opts.ConnectionLimit != 0 {
		q["connection_limit"] = strconv.Itoa(opts.ConnectionLimit)
	}
	if opts.Marker != "" {
		q["marker"] = opts.Marker
	}
	if opts.Limit != 0 {
		q["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.SortKey != "" {
		q["sort_key"] = opts.SortKey
	}
	if opts.SortDir != "" {
		q["sort_dir"] = opts.SortDir
	}

	u := rootURL(c) + utils.BuildQuery(q)

	return pagination.NewPager(c, u, func(r pagination.LastHTTPResponse) pagination.Page {
		return VIPPage{pagination.LinkedPageBase{LastHTTPResponse: r}}
	})
}

// CreateOpts contains all the values needed to create a new virtual IP.
type CreateOpts struct {
	// Required. Human-readable name for the VIP. Does not have to be unique.
	Name string

	// Required. The network on which to allocate the VIP's address. A tenant can
	// only create VIPs on networks authorized by policy (e.g. networks that
	// belong to them or networks that are shared).
	SubnetID string

	// Required. The protocol - can either be TCP, HTTP or HTTPS.
	Protocol string

	// Required. The port on which to listen for client traffic.
	ProtocolPort int

	// Required. The ID of the pool with which the VIP is associated.
	PoolID string

	// Required for admins. Indicates the owner of the VIP.
	TenantID string

	// Optional. The IP address of the VIP.
	Address string

	// Optional. Human-readable description for the VIP.
	Description string

	// Optional. Omit this field to prevent session persistence.
	Persistence *SessionPersistence

	// Optional. The maximum number of connections allowed for the VIP.
	ConnLimit *int

	// Optional. The administrative state of the VIP. A valid value is true (UP)
	// or false (DOWN).
	AdminStateUp *bool
}

var (
	errNameRequired         = fmt.Errorf("Name is required")
	errSubnetIDRequried     = fmt.Errorf("SubnetID is required")
	errProtocolRequired     = fmt.Errorf("Protocol is required")
	errProtocolPortRequired = fmt.Errorf("Protocol port is required")
	errPoolIDRequired       = fmt.Errorf("PoolID is required")
)

// Create is an operation which provisions a new virtual IP based on the
// configuration defined in the CreateOpts struct. Once the request is
// validated and progress has started on the provisioning process, a
// CreateResult will be returned.
//
// Please note that the PoolID should refer to a pool that is not already
// associated with another vip. If the pool is already used by another vip,
// then the operation will fail with a 409 Conflict error will be returned.
//
// Users with an admin role can create VIPs on behalf of other tenants by
// specifying a TenantID attribute different than their own.
func Create(c *gophercloud.ServiceClient, opts CreateOpts) CreateResult {
	var res CreateResult

	// Validate required opts
	if opts.Name == "" {
		res.Err = errNameRequired
		return res
	}
	if opts.SubnetID == "" {
		res.Err = errSubnetIDRequried
		return res
	}
	if opts.Protocol == "" {
		res.Err = errProtocolRequired
		return res
	}
	if opts.ProtocolPort == 0 {
		res.Err = errProtocolPortRequired
		return res
	}
	if opts.PoolID == "" {
		res.Err = errPoolIDRequired
		return res
	}

	type vip struct {
		Name         string              `json:"name"`
		SubnetID     string              `json:"subnet_id"`
		Protocol     string              `json:"protocol"`
		ProtocolPort int                 `json:"protocol_port"`
		PoolID       string              `json:"pool_id"`
		Description  *string             `json:"description,omitempty"`
		TenantID     *string             `json:"tenant_id,omitempty"`
		Address      *string             `json:"address,omitempty"`
		Persistence  *SessionPersistence `json:"session_persistence,omitempty"`
		ConnLimit    *int                `json:"connection_limit,omitempty"`
		AdminStateUp *bool               `json:"admin_state_up,omitempty"`
	}

	type request struct {
		VirtualIP vip `json:"vip"`
	}

	reqBody := request{VirtualIP: vip{
		Name:         opts.Name,
		SubnetID:     opts.SubnetID,
		Protocol:     opts.Protocol,
		ProtocolPort: opts.ProtocolPort,
		PoolID:       opts.PoolID,
		Description:  gophercloud.MaybeString(opts.Description),
		TenantID:     gophercloud.MaybeString(opts.TenantID),
		Address:      gophercloud.MaybeString(opts.Address),
		ConnLimit:    opts.ConnLimit,
		AdminStateUp: opts.AdminStateUp,
	}}

	if opts.Persistence != nil {
		reqBody.VirtualIP.Persistence = opts.Persistence
	}

	_, res.Err = perigee.Request("POST", rootURL(c), perigee.Options{
		MoreHeaders: c.Provider.AuthenticatedHeaders(),
		ReqBody:     &reqBody,
		Results:     &res.Resp,
		OkCodes:     []int{201},
	})

	return res
}
