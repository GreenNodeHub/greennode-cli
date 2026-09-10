// Package operation builds descriptor-driven commands.
package operation

// PathParam binds a required flag to a URL placeholder; validation precedes escaping.
type PathParam struct {
	Placeholder string
	Flag        string
	Usage       string
	Validate    func(value, flag string) error
	// Secret requires masking the resolved path value in previews, prompts, and debug output.
	Secret bool
}

// QueryKind selects query parsing and validation.
type QueryKind uint8

const (
	// QueryString passes the flag value through unchanged.
	QueryString QueryKind = iota
	// QueryInteger requires a base-10 32-bit integer, optionally bounded by Minimum.
	QueryInteger
	// QueryBoolean requires strconv.ParseBool and normalizes the wire value to "true"/"false".
	QueryBoolean
	// QueryID additionally validates the value with validator.ValidateID.
	QueryID
	// QueryObject requires a JSON object and re-encodes it compactly on the wire.
	QueryObject
)

// QueryParam binds one query-string parameter to a CLI flag.
type QueryParam struct {
	WireName string
	Flag     string
	Usage    string
	Required bool
	Kind     QueryKind
	// Minimum, when > 0, additionally bounds a QueryInteger value.
	Minimum int64
	// Default applies before required-value validation.
	Default string
	// EncodeError customizes query-object encoding errors.
	EncodeError func(flag string, err error) error
	// ObjectStyle selects query-object error wording.
	ObjectStyle QueryObjectStyle
}

// QueryObjectStyle selects compatible decode-error wording.
type QueryObjectStyle uint8

const (
	// QueryObjectJSONObject produces vlb's wording: "invalid %s JSON object: ...".
	QueryObjectJSONObject QueryObjectStyle = iota
	// QueryObjectJSON produces vdb's wording: "invalid %s JSON: ...".
	QueryObjectJSON
)

// BodyKind selects the top-level JSON shape ParseBody requires.
type BodyKind uint8

const (
	// NoBody means the operation takes no request body at all.
	NoBody BodyKind = iota
	// ObjectBody requires a JSON object.
	ObjectBody
	// ArrayBody requires an array; required fields apply to each object item.
	ArrayBody
	// OptionalObjectBody allows omission, otherwise requires an object.
	OptionalObjectBody
	// OptionalJSONBody accepts omission or any explicit JSON value for unpublished schemas.
	OptionalJSONBody
)

// BodyStyle selects compatible body-validation error wording.
type BodyStyle uint8

const (
	// BodyStyleJSONBody reports "invalid JSON body".
	BodyStyleJSONBody BodyStyle = iota
	// BodyStyleJSONObject produces vlb/vcr's wording: "invalid body JSON
	// object: ...".
	BodyStyleJSONObject
	// BodyStyleJSON produces vdb's wording: "invalid body JSON: ...".
	BodyStyleJSON
)

// BodyContract describes a Descriptor's request-body requirement.
type BodyContract struct {
	Kind  BodyKind
	Usage string
	Style BodyStyle
	// Name labels missing-field errors.
	Name string
	// RequiredFields are checked on a top-level ObjectBody.
	RequiredFields []string
	// RequiredItemFields validates object items when StrictArrayItems is set.
	RequiredItemFields []string
	// StrictArrayItems requires every array element to be an object.
	StrictArrayItems bool
}

// Descriptor declares an operation's flags, execution policy, and response contract.
type Descriptor struct {
	Parent string
	Use    string
	Short  string
	// OperationID links the operation to its published contract.
	OperationID string
	Method      string
	Path        string

	Paths   []PathParam
	Queries []QueryParam
	Body    *BodyContract

	Mutation    bool
	Destructive bool
	// Download uses RunDownload and requires dry-run and force flags.
	Download bool

	// EmptySuccess lists statuses permitting empty bodies.
	EmptySuccess []int
	// Status and AlternateStatuses describe success codes; ResponseBody declares a JSON body.
	Status            int
	AlternateStatuses []int
	ResponseBody      bool

	// SecretResponse requires output masking unless show-secret is set; always mask logs.
	SecretResponse bool

	// SecretPathParams marks secrets for services that derive path bindings from templates.
	SecretPathParams []string
	// RawResponse selects raw-string transport.
	RawResponse bool
	// NoPortalUserID and UserType describe operation-specific vDB header requirements.
	NoPortalUserID bool
	UserType       bool

	// Extra carries service-specific data interpreted only by that service's hooks.
	Extra any
}

// HasEmptyStatus checks EmptySuccess membership.
func (d Descriptor) HasEmptyStatus(statusCode int) bool {
	for _, allowed := range d.EmptySuccess {
		if statusCode == allowed {
			return true
		}
	}
	return false
}

// HasAlternateStatus reports whether statusCode is one of d.AlternateStatuses.
func (d Descriptor) HasAlternateStatus(statusCode int) bool {
	for _, allowed := range d.AlternateStatuses {
		if statusCode == allowed {
			return true
		}
	}
	return false
}
