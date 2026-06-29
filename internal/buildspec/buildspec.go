// Package buildspec is the KIDS build-spec schema + validating loader
// (VSL T0a.1, CQ9): the declarative, diffable git source of a KIDS BUILD (#9.6),
// kids/<pkg>.build.json. It is the human-readable form of a package definition —
// component list, Required Builds, the environment-check routine, and the ICR
// list — that `v pkg build` consumes to produce a transport global / .KID.
// See msl-vsl-coordination-implementation-plan.md §7.2.
package buildspec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Required-build install actions (KIDS BUILD #9.6 REQUIRED BUILD #11 multiple;
// architecture §3.1).
const (
	ActionWarn         = "WARNING ONLY"                 // warn, continue
	ActionLeaveGlobal  = "DON'T INSTALL, LEAVE GLOBAL"  // block unless prerequisite present; keep its global
	ActionRemoveGlobal = "DON'T INSTALL, REMOVE GLOBAL" // block unless prerequisite present; remove its global
)

var validActions = map[string]bool{ActionWarn: true, ActionLeaveGlobal: true, ActionRemoveGlobal: true}

// RequiredBuildActionCode maps a Required-Build action to its #9.611 ACTION
// set-of-codes value (0 = warn, 1 = don't install/remove global, 2 = don't
// install/leave global).
var RequiredBuildActionCode = map[string]int{
	ActionWarn:         0,
	ActionRemoveGlobal: 1,
	ActionLeaveGlobal:  2,
}

// Spec is one KIDS build definition (kids/<pkg>.build.json).
type Spec struct {
	Package        string          `json:"package"`         // NAMESPACE (e.g. ZZSKEL)
	Version        string          `json:"version"`         // e.g. 1.0
	Patch          string          `json:"patch,omitempty"` // e.g. 1 (optional)
	Components     Components      `json:"components"`
	RequiredBuilds []RequiredBuild `json:"requiredBuilds,omitempty"`
	EnvCheck       string          `json:"envCheck,omitempty"`    // environment-check routine (bare name, XPDENV)
	PreInstall     string          `json:"preInstall,omitempty"`  // pre-install routine entryref (TAG^RTN), run before component filing
	PostInstall    string          `json:"postInstall,omitempty"` // post-install routine entryref (TAG^RTN), run after component filing
	ICRs           []ICR           `json:"icrs,omitempty"`        // DBIA/ICR agreements the package relies on

	// AllowLongNames opts the build out of the legacy 8-char SAC routine-name
	// convention, raising the cap to the M engine limit (RoutineNameMaxLong).
	// The M-naming character rules still bind. This is a deliberate, committed
	// policy choice — see the org "modern routine-name length policy" ADR — and
	// is appropriate only for modern YDB/IRIS targets (it forgoes legacy
	// GT.M/Caché %RO portability and formal SAC/national release without a
	// later rename).
	AllowLongNames bool `json:"allowLongNames,omitempty"`
}

// Components is the BUILD component list (#9.6). Each slice is omitempty so a
// spec lists only what it ships.
type Components struct {
	Routines             []string           `json:"routines,omitempty"`
	Files                []FileComp         `json:"files,omitempty"`
	Options              []OptionComp       `json:"options,omitempty"`              // #19 OPTION KRN components (B.1)
	Keys                 []KeyComp          `json:"keys,omitempty"`                 // #19.1 SECURITY KEY KRN components (B.1)
	Parameters           []string           `json:"parameters,omitempty"`           // XPAR parameter names (reference only)
	ParameterDefinitions []ParamDef         `json:"parameterDefinitions,omitempty"` // XPAR #8989.51 PARAMETER DEFINITION components (shipped as data)
	Protocols            []ProtocolComp     `json:"protocols,omitempty"`            // #101 PROTOCOL KRN components (B.1)
	Templates            []string           `json:"templates,omitempty"`
	RPCs                 []RPCComp          `json:"rpcs,omitempty"`            // #8994 REMOTE PROCEDURE KRN components (B.1)
	MailGroups           []MailGroupComp    `json:"mailGroups,omitempty"`      // #3.8 MAIL GROUP KRN components (B.1)
	ListTemplates        []ListTemplateComp `json:"listTemplates,omitempty"`   // #409.61 LIST TEMPLATE KRN components (B.1)
	HelpFrames           []HelpFrameComp    `json:"helpFrames,omitempty"`      // #9.2 HELP FRAME KRN components (B.1)
	HL7Applications      []HL7AppComp       `json:"hl7Applications,omitempty"` // #771 HL7 APPLICATION PARAMETER KRN components (B.1)
	HLOApplications      []HLOAppComp       `json:"hloApplications,omitempty"` // #779.2 HLO APPLICATION REGISTRY KRN components (B.1)
	LogicalLinks         []LogicalLinkComp  `json:"logicalLinks,omitempty"`    // #870 HL LOGICAL LINK KRN components (B.1)
}

func (c Components) empty() bool {
	return len(c.Routines) == 0 && len(c.Files) == 0 && len(c.Options) == 0 &&
		len(c.Keys) == 0 && len(c.Parameters) == 0 && len(c.ParameterDefinitions) == 0 &&
		len(c.Protocols) == 0 && len(c.Templates) == 0 && len(c.RPCs) == 0 &&
		len(c.MailGroups) == 0 && len(c.ListTemplates) == 0 && len(c.HelpFrames) == 0 &&
		len(c.HL7Applications) == 0 && len(c.HLOApplications) == 0 && len(c.LogicalLinks) == 0
}

// unsupported returns the JSON names of populated component slices the build
// path has no emitter for. The slices exist in the schema as forward-looking
// placeholders, but `v pkg build` emits only routines, files, and
// parameterDefinitions today — so declaring one of these would silently produce
// an incomplete build that installs "successfully" (coverage-analysis F1).
// Validate rejects that rather than drop it. `Parameters` is intentionally
// excluded: it is reference-only metadata (like ICRs), not a shipped component —
// to actually ship a parameter, use ParameterDefinitions.
func (c Components) unsupported() []string {
	var u []string
	for _, e := range []struct {
		name string
		n    int
	}{
		{"templates", len(c.Templates)},
	} {
		if e.n > 0 {
			u = append(u, e.name)
		}
	}
	return u
}

// ParamDef is an XPAR PARAMETER DEFINITION (#8989.51) shipped as a KIDS KRN
// component — the build creates the definition (not a value) at install time.
// DataType/Entity are human names resolved to their #8989.51 codes / #8989.518
// IENs by ParamDataTypeCode / ParamEntityIEN.
type ParamDef struct {
	Name        string        `json:"name"`                  // #8989.51 .01 NAME (uppercase, e.g. "VSL GREETING")
	DisplayText string        `json:"displayText,omitempty"` // #8989.51 .02 DISPLAY TEXT
	DataType    string        `json:"dataType,omitempty"`    // value data type; default "free text"
	Entities    []ParamEntity `json:"entities,omitempty"`    // ALLOWABLE ENTITIES (#8989.513) — where the param may be set
}

// ParamEntity is one ALLOWABLE ENTITIES row of a ParamDef: which entity the
// parameter may be set at, and its precedence.
type ParamEntity struct {
	Entity     string `json:"entity"`               // entity abbreviation: SYS, USR, PKG, …
	Precedence int    `json:"precedence,omitempty"` // #8989.513 .01 PRECEDENCE; default 1
}

// OptionComp is a #19 OPTION shipped as a KIDS KRN component (B.1). The build
// files the option definition (never a value); Type is a human option-type name
// resolved to its #19 field 4 (TYPE) set-of-codes value by OptionTypeCode. A
// run-routine option requires Routine; an action option typically sets EntryAction.
type OptionComp struct {
	Name        string `json:"name"`                  // #19 .01 NAME (uppercase, e.g. "ZZOPT RUN ROUTINE")
	MenuText    string `json:"menuText,omitempty"`    // #19 field 1 MENU TEXT
	Type        string `json:"type"`                  // option type: menu | run routine | action | ...
	Routine     string `json:"routine,omitempty"`     // #19 field 25 ROUTINE entryref (required for "run routine")
	EntryAction string `json:"entryAction,omitempty"` // #19 field 20 ENTRY ACTION (M code)
	ExitAction  string `json:"exitAction,omitempty"`  // #19 field 15 EXIT ACTION (M code)
}

// RPCComp is a #8994 REMOTE PROCEDURE shipped as a KIDS KRN component (B.1). The
// DD-required fields are Name (.01) + Routine (.03) + ReturnType (.04); Tag (.02)
// is functionally required for the RPC to run. ReturnType is a human name resolved
// to its #8994 field .04 set-of-codes value by RPCReturnTypeCode (default "single
// value").
type RPCComp struct {
	Name            string         `json:"name"`                      // #8994 .01 NAME (uppercase, e.g. "ZZRPC ECHO")
	Tag             string         `json:"tag,omitempty"`             // #8994 .02 TAG — the M entry tag
	Routine         string         `json:"routine"`                   // #8994 .03 ROUTINE — the M routine
	ReturnType      string         `json:"returnType,omitempty"`      // value type; default "single value"
	InputParameters []RPCParamComp `json:"inputParameters,omitempty"` // #8994.02 INPUT PARAMETER multiple
}

// RPCParamComp is one #8994.02 INPUT PARAMETER of an RPC: its name, type, max data
// length, required flag, sequence, and an optional DESCRIPTION. Type defaults to
// "literal"; sequence defaults to the parameter's 1-based position.
type RPCParamComp struct {
	Name        string   `json:"name"`                  // #8994.02 .01 INPUT PARAMETER
	Type        string   `json:"type,omitempty"`        // literal (default) | list | word processing | reference
	MaxLength   int      `json:"maxLength,omitempty"`   // #8994.02 .03 MAXIMUM DATA LENGTH
	Required    bool     `json:"required,omitempty"`    // #8994.02 .04 REQUIRED?
	Sequence    int      `json:"sequence,omitempty"`    // #8994.02 .05 SEQUENCE NUMBER (default = position)
	Description []string `json:"description,omitempty"` // #8994.02 DESCRIPTION word-processing lines
}

// RPCParamTypeCode maps a human RPC-parameter-type name to its #8994.02 field .02
// (PARAMETER TYPE) set-of-codes value. National constants.
var RPCParamTypeCode = map[string]string{
	"literal":         "1",
	"list":            "2",
	"word processing": "3",
	"reference":       "4",
}

// RPCReturnTypeCode maps a human return-value-type name to its #8994 field .04
// (RETURN VALUE TYPE) set-of-codes value. National constants.
var RPCReturnTypeCode = map[string]string{
	"single value":    "1",
	"array":           "2",
	"word processing": "3",
	"global array":    "4",
	"global instance": "5",
}

// ProtocolComp is a #101 PROTOCOL shipped as a KIDS KRN component (B.1). Type is a
// human option-type name resolved to its #101 field 4 (TYPE) set-of-codes value by
// ProtocolTypeCode. The build files a base protocol (the #101.01 ITEM multiple +
// extended menu-actions are a follow-up).
type ProtocolComp struct {
	Name        string             `json:"name"`                  // #101 .01 NAME (uppercase, e.g. "ZZPROTO ACTION")
	ItemText    string             `json:"itemText,omitempty"`    // #101 field 1 ITEM TEXT
	Type        string             `json:"type"`                  // protocol type: action | extended action | menu | event driver | ...
	EntryAction string             `json:"entryAction,omitempty"` // #101 field 20 ENTRY ACTION (M code)
	ExitAction  string             `json:"exitAction,omitempty"`  // #101 field 15 EXIT ACTION (M code)
	Items       []ProtocolItemComp `json:"items,omitempty"`       // #101.01 ITEM multiple — child protocols of a menu
}

// ProtocolItemComp is one ITEM of a menu PROTOCOL (#101.01): the NAME of the child
// protocol to attach and its display sequence. The install re-points the child by
// name (KIDS "^" resolver node), so cross-build references work.
type ProtocolItemComp struct {
	Name     string `json:"name"`               // child PROTOCOL .01 NAME (#101 pointer target)
	Sequence int    `json:"sequence,omitempty"` // #101.01 .03 SEQUENCE (default = position)
}

// ProtocolTypeCode maps a human protocol-type name to its #101 field 4 (TYPE)
// set-of-codes value. Grounded in the live ^DD(101,4) set string (these codes are
// #101's OWN — they differ from #19 OPTION's). National constants.
var ProtocolTypeCode = map[string]string{
	"action":           "A",
	"menu":             "M",
	"protocol":         "O",
	"protocol menu":    "Q",
	"limited protocol": "L",
	"extended action":  "X",
	"dialog":           "D",
	"term":             "T",
	"event driver":     "E",
	"subscriber":       "S",
}

// KeyComp is a #19.1 SECURITY KEY shipped as a KIDS KRN component (B.1). A key is
// a named token holders are granted; the build files the key by name (its optional
// word-processing DESCRIPTION is a follow-up).
type KeyComp struct {
	Name        string   `json:"name"`                  // #19.1 .01 NAME (uppercase, e.g. "ZZKEY MANAGER")
	Description []string `json:"description,omitempty"` // #19.1 DESCRIPTION word-processing lines (optional)
}

// MailGroupComp is a #3.8 MAIL GROUP shipped as a KIDS KRN component (B.1). Type is
// a human name resolved to the #3.8 field 4 (TYPE) set-of-codes value by
// MailGroupTypeCode (default "public"). The build files the group definition only —
// KIDS ships mail groups MEMBER-less (the #200-pointer member list is site-local),
// and the word-processing DESCRIPTION is deferred (its header carries a volatile
// last-edited date that would defeat the deterministic-build invariant).
type MailGroupComp struct {
	Name                string   `json:"name"`                          // #3.8 .01 NAME (uppercase, e.g. "ZZMG ALERTS")
	Type                string   `json:"type,omitempty"`                // mail-group type: public (default) | private
	AllowSelfEnrollment bool     `json:"allowSelfEnrollment,omitempty"` // #3.8 field 7 ALLOW SELF ENROLLMENT?
	Description         []string `json:"description,omitempty"`         // #3.8 DESCRIPTION word-processing lines (optional)
}

// MailGroupTypeCode maps a human mail-group-type name to its #3.8 field 4 (TYPE)
// set-of-codes value. Grounded in the live ^DD(3.8,4) set string. National constants.
var MailGroupTypeCode = map[string]string{
	"public":  "PU",
	"private": "PR",
}

// ListTemplateComp is a #409.61 LIST TEMPLATE (List Manager screen) shipped as a
// KIDS KRN component (B.1). The build files the screen definition: a name, screen
// geometry (margins, defaulted), the action PROTOCOL MENU (a #101 pointer by name),
// the screen title, and the List Manager callback codes (header/init/exit/help M
// code + the display ARRAY global). All plain strings — no compiled structure.
type ListTemplateComp struct {
	Name         string `json:"name"`                   // #409.61 .01 NAME (uppercase, e.g. "ZZLM PATIENTS")
	ScreenTitle  string `json:"screenTitle,omitempty"`  // .11 SCREEN TITLE
	ProtocolMenu string `json:"protocolMenu,omitempty"` // .1 PROTOCOL MENU — #101 action-menu name
	RightMargin  int    `json:"rightMargin,omitempty"`  // .04 RIGHT MARGIN (default 80)
	TopMargin    int    `json:"topMargin,omitempty"`    // .05 TOP MARGIN (default 3)
	BottomMargin int    `json:"bottomMargin,omitempty"` // .06 BOTTOM MARGIN (default 20)
	HeaderCode   string `json:"headerCode,omitempty"`   // HEADER CODE (M code, node "HDR")
	EntryCode    string `json:"entryCode,omitempty"`    // ENTRY CODE (M code, node "INIT")
	ExitCode     string `json:"exitCode,omitempty"`     // EXIT CODE (M code, node "FNL")
	HelpCode     string `json:"helpCode,omitempty"`     // HELP CODE (M code, node "HLP")
	ArrayName    string `json:"arrayName,omitempty"`    // ARRAY NAME (display global ref, node "ARRAY")
}

// HelpFrameComp is a #9.2 HELP FRAME shipped as a KIDS KRN component (B.1). The
// build files the frame: a name, a one-line HEADER summary, and the TEXT body (a
// word-processing field — the help content). HELP FRAME names allow spaces AND
// hyphens (e.g. "YS-PHY-EXAM-NORM"), unlike the space-only OPTION/RPC names.
type HelpFrameComp struct {
	Name   string   `json:"name"`             // #9.2 .01 NAME (3–30 chars, uppercase, hyphens/spaces)
	Header string   `json:"header,omitempty"` // #9.2 field 1 HEADER — one-line summary
	Text   []string `json:"text,omitempty"`   // #9.2 field 2 TEXT — the help body, one string per line
}

// HL7AppComp is a #771 HL7 APPLICATION PARAMETER shipped as a KIDS KRN component
// (B.1) — the registration of an application that sends/receives HL7 messages. The
// build files the entry ACTIVE (a shipped registration is always active);
// CountryCode defaults to "USA". HL7 application names allow spaces AND underscores
// (e.g. "VISTA_VTS"), unlike the space-only OPTION/RPC names. (The HLO registry
// #779.2 and logical link #870 are follow-ups.)
type HL7AppComp struct {
	Name        string `json:"name"`                  // #771 .01 NAME (uppercase, spaces/underscores)
	Facility    string `json:"facility,omitempty"`    // #771 field 3 FACILITY NAME
	CountryCode string `json:"countryCode,omitempty"` // #771 field 7 COUNTRY CODE (default "USA")
}

// LogicalLinkComp is a #870 HL LOGICAL LINK shipped as a KIDS KRN component (B.1) —
// an HL7 communication endpoint. LLPType defaults to "TCP" (shipped as the external
// #869.1 value, resolved to its IEN at install). When Port (and/or ServiceType) is
// present the build also files the 400-node; ServiceType defaults to "C"
// (CLIENT/SENDER). A skeleton link with no TCP params is valid.
//
// There is intentionally NO DNS domain / IP address field: the #870 install does
// not transport the network endpoint (it is site-specific config the install drops
// — see kids.LogicalLink). v-pkg ships the link definition; the site configures the
// endpoint. (The DESCRIPTION word-processing field is a follow-up.)
type LogicalLinkComp struct {
	Name        string   `json:"name"`                  // #870 .01 NODE (3..10 chars)
	LLPType     string   `json:"llpType,omitempty"`     // #870 field 2 LLP TYPE (default "TCP")
	Port        string   `json:"port,omitempty"`        // #870 400.02 TCP/IP PORT
	ServiceType string   `json:"serviceType,omitempty"` // #870 400.03 TCP/IP SERVICE TYPE (C/S/M, default "C")
	Description []string `json:"description,omitempty"` // #870 DESCRIPTION word-processing lines (optional)
}

// HLOAppComp is a #779.2 HLO APPLICATION REGISTRY shipped as a KIDS KRN component
// (B.1) — the HL7-Optimized counterpart to #771. It registers an application and
// maps the HL7 message types it handles (the messageTypes multiple) to action
// routines. The emitter ships each message-type entry with its computed
// cross-references.
type HLOAppComp struct {
	Name         string          `json:"name"`                   // #779.2 .01 APPLICATION NAME (3..60 chars)
	MessageTypes []HLOMsgTypComp `json:"messageTypes,omitempty"` // #779.21 MESSAGE TYPE ACTIONS
}

// HLOMsgTypComp is one MESSAGE TYPE ACTIONS entry (#779.21) of an HLO application:
// the HL7 message type/event it handles and the action routine that processes it.
type HLOMsgTypComp struct {
	MessageType   string `json:"messageType"`             // #779.21 .01 HL7 MESSAGE TYPE
	Event         string `json:"event"`                   // #779.21 .02 HL7 EVENT
	ActionTag     string `json:"actionTag,omitempty"`     // #779.21 .04 ACTION TAG
	ActionRoutine string `json:"actionRoutine,omitempty"` // #779.21 .05 ACTION ROUTINE
	Version       string `json:"version,omitempty"`       // #779.21 .06 HL7 VERSION
}

// OptionTypeCode maps a human option-type name to its #19 field 4 (TYPE)
// set-of-codes value. Grounded in the live ^DD(19,4) set string (Kernel Menu
// Manager). These codes are national constants — portable across YDB and IRIS.
var OptionTypeCode = map[string]string{
	"action":          "A",
	"edit":            "E",
	"inquire":         "I",
	"menu":            "M",
	"print":           "P",
	"run routine":     "R",
	"protocol":        "O",
	"protocol menu":   "Q",
	"extended action": "X",
	"server":          "S",
}

// ParamDataTypeCode maps a human value-data-type name to the #8989.51 field 1.1
// (VALUE DATA TYPE) set-of-codes value. Free text is the default; the rest are
// accepted for completeness.
var ParamDataTypeCode = map[string]string{
	"free text":       "F",
	"numeric":         "N",
	"yes/no":          "Y",
	"set of codes":    "S",
	"date/time":       "D",
	"pointer":         "P",
	"word processing": "W",
}

// ParamEntityIEN maps a parameter-entity abbreviation to its #8989.518
// (PARAMETER ENTITY) IEN. #8989.518 is DINUM'd to the pointed-to file number, so
// these IENs are national constants — portable across YDB and IRIS VistA.
var ParamEntityIEN = map[string]string{
	"DEV": "3.5",  // DEVICE
	"DIV": "4",    // DIVISION
	"SYS": "4.2",  // SYSTEM (KERNEL SYSTEM PARAMETERS)
	"PKG": "9.4",  // PACKAGE
	"LOC": "44",   // LOCATION
	"SRV": "49",   // SERVICE
	"USR": "200",  // USER (NEW PERSON)
	"CLS": "8930", // CLASS
}

// FileComp is a FileMan FILE component shipped as a KIDS data-dictionary export.
// For VSL M3.T1 the supported shape is a brand-new throwaway file with a single
// free-text .01 NAME field (the minimal transform-invariant DD); Number is a
// local/test file number (999000-999999) and GlobalRoot is its data global
// (defaulting to ^DIZ(<file>,).
type FileComp struct {
	Number     float64     `json:"number"`
	Name       string      `json:"name,omitempty"`
	GlobalRoot string      `json:"globalRoot,omitempty"` // data global, e.g. "^DIZ(999000,"; default derived from Number
	Fields     []FieldSpec `json:"fields,omitempty"`     // fields beyond the implicit .01 NAME (empty = single-.01 file)
}

// FieldSpec is one FileMan field beyond the implicit free-text .01 NAME, declared
// in a FILE component. Type selects the DD grammar; Node/Piece are the storage
// location ("<Node>;<Piece>"). Type-specific knobs (Width/Decimals/Min/Max for
// numeric, Codes for set-of-codes, PointTo/PointRoot for pointer, MaxLen for free
// text, Time for date) are validated against the declared Type.
type FieldSpec struct {
	Number   float64 `json:"number"`             // field number (> .01)
	Label    string  `json:"label"`              // field LABEL (uppercase)
	Type     string  `json:"type"`               // free text | numeric | date | set of codes | pointer
	Node     int     `json:"node,omitempty"`     // storage node (default 0)
	Piece    int     `json:"piece"`              // storage piece (≥1; (0;1) is reserved for the .01)
	Required bool    `json:"required,omitempty"` // R attribute
	Help     string  `json:"help,omitempty"`     // reader help prompt

	MaxLen int `json:"maxLen,omitempty"` // free text: max length (0 → default 30)

	Width    int      `json:"width,omitempty"`    // numeric: print width (NJ<width>,<decimals>)
	Decimals int      `json:"decimals,omitempty"` // numeric: decimal places
	Min      *float64 `json:"min,omitempty"`      // numeric: lower bound
	Max      *float64 `json:"max,omitempty"`      // numeric: upper bound

	Time bool `json:"time,omitempty"` // date: allow a time component

	Codes []SetCodeSpec `json:"codes,omitempty"` // set of codes: value list

	PointTo   float64 `json:"pointTo,omitempty"`   // pointer: pointed-to file number
	PointRoot string  `json:"pointRoot,omitempty"` // pointer: pointed-to global root
}

// SetCodeSpec is one internal:external value of a set-of-codes field.
type SetCodeSpec struct {
	Internal string `json:"internal"`
	External string `json:"external"`
}

// validFieldTypes is the set of FileMan field types the multi-field DD emitter
// supports today (the five grounded scalar shapes).
var validFieldTypes = map[string]bool{
	"free text":    true,
	"numeric":      true,
	"date":         true,
	"set of codes": true,
	"pointer":      true,
}

// freeTextMaxLen caps a free-text field's declared max length (FileMan stores a
// field value on a single global node piece; 245 is the practical ceiling).
const freeTextMaxLen = 245

// Local/test FileMan file-number range (VA-reserved 999000-999999). A v-pkg
// DD-install ships only throwaway files in this band — never a national file.
const (
	LocalFileMin = 999000
	LocalFileMax = 999999
)

// Routine-name length caps. RoutineNameMaxStd is the legacy SAC/DSM-era
// convention (a 2-4 char namespace + remainder, 8 total). RoutineNameMaxLong is
// the modern M engine limit honored by YottaDB and IRIS (31 significant
// characters) — used when a spec sets allowLongNames. KIDS itself (transport
// global, string-subscript keyed) and XINDEX impose no name-length limit; the 8
// is a convention, the 31 is the real engine bound.
const (
	RoutineNameMaxStd  = 8
	RoutineNameMaxLong = 31
)

// RequiredBuild is a KIDS Required Build (#11) — a prerequisite build + the
// action KIDS takes when it is absent.
type RequiredBuild struct {
	Name   string `json:"name"`   // NAMESPACE*VERSION[*PATCH]
	Action string `json:"action"` // one of the Action* constants
}

// ICR is a DBIA/Integration Control Registration the package depends on.
type ICR struct {
	Number    int    `json:"number"`
	Name      string `json:"name,omitempty"`
	Custodian string `json:"custodian,omitempty"`
}

// InstallName is the KIDS install name NAMESPACE*VERSION[*PATCH].
func (s *Spec) InstallName() string {
	n := s.Package + "*" + s.Version
	if s.Patch != "" {
		n += "*" + s.Patch
	}
	return n
}

var (
	reNamespace = regexp.MustCompile(`^[A-Z%][A-Z0-9]*$`)
	reVersion   = regexp.MustCompile(`^\d+\.\d+$`)
	rePatch     = regexp.MustCompile(`^\d+$`)
	reRoutine   = regexp.MustCompile(`^%?[A-Z][A-Z0-9]*$`)                  // M routine name
	reReqBuild  = regexp.MustCompile(`^[A-Z%][A-Z0-9]*\*\d+\.\d+(\*\d+)?$`) // NS*VER[*PATCH]
	reParamName = regexp.MustCompile(`^[A-Z][A-Z0-9 ]*[A-Z0-9]$|^[A-Z]$`)   // #8989.51 NAME (uppercase, internal spaces)
	reEntryName = regexp.MustCompile(`^[A-Z][A-Z0-9 ]*[A-Z0-9]$|^[A-Z]$`)   // #19 OPTION .01 NAME (uppercase, internal spaces)
	reHelpName  = regexp.MustCompile(`^[A-Z][A-Z0-9 -]*[A-Z0-9]$`)          // #9.2 HELP FRAME .01 NAME (uppercase, spaces AND hyphens)
	reHL7Name   = regexp.MustCompile(`^[A-Z][A-Z0-9 _]*[A-Z0-9]$`)          // #771 HL7 APPLICATION .01 NAME (uppercase, spaces AND underscores)
	reLinkName  = regexp.MustCompile(`^[A-Z][A-Z0-9 ._-]*$`)                // #870 HL LOGICAL LINK .01 NODE (no leading punctuation; spaces/dots/hyphens/underscores)
	reSvcType   = regexp.MustCompile(`^[CSM]$`)                             // #870 400.03 SERVICE TYPE set code
	reHL7Code   = regexp.MustCompile(`^[A-Z][A-Z0-9]*$`)                    // #779.21 HL7 MESSAGE TYPE / EVENT code (uppercase alnum)
	reFileName  = regexp.MustCompile(`^[A-Z][A-Z0-9 ]*[A-Z0-9]$|^[A-Z]$`)   // FileMan FILE .01 name (uppercase, internal spaces)
	reGlobalRt  = regexp.MustCompile(`^\^%?[A-Z][A-Z0-9]*\(.*,$`)           // open global root, e.g. ^DIZ(999000,
	reLabel     = regexp.MustCompile(`^[A-Z][A-Z0-9]*$`)                    // M line label (entryref tag)
)

// Load reads and validates a build spec from a kids/<pkg>.build.json file.
func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("buildspec: read %s: %w", path, err)
	}
	return Parse(data)
}

// Parse decodes a build spec from JSON (rejecting unknown fields — a typo'd key
// is an error, not a silent drop) and validates it.
func Parse(data []byte) (*Spec, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var s Spec
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("buildspec: parse: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

// Validate checks the spec is a usable KIDS build definition.
func (s *Spec) Validate() error {
	if !reNamespace.MatchString(s.Package) {
		return fmt.Errorf("buildspec: package %q is not a valid VistA namespace", s.Package)
	}
	if !reVersion.MatchString(s.Version) {
		return fmt.Errorf("buildspec: version %q must be MAJOR.MINOR (e.g. 1.0)", s.Version)
	}
	if s.Patch != "" && !rePatch.MatchString(s.Patch) {
		return fmt.Errorf("buildspec: patch %q must be numeric", s.Patch)
	}
	if s.Components.empty() {
		return fmt.Errorf("buildspec: %s has no components — a build must ship at least one", s.InstallName())
	}
	if u := s.Components.unsupported(); len(u) > 0 {
		return fmt.Errorf("buildspec: %s declares component type(s) [%s] that v-pkg's build path "+
			"cannot emit yet — a build would silently omit them. Remove them, or extend v-pkg "+
			"(see docs/proposals/v-pkg-kids-coverage-analysis.md, Track B). Emittable today: "+
			"routines, files, parameterDefinitions, requiredBuilds", s.InstallName(), strings.Join(u, ", "))
	}
	maxName := RoutineNameMaxStd
	if s.AllowLongNames {
		maxName = RoutineNameMaxLong
	}
	if err := validateRoutines(s.Components.Routines, maxName); err != nil {
		return err
	}
	if err := validateParamDefs(s.Components.ParameterDefinitions); err != nil {
		return err
	}
	if err := validateOptions(s.Components.Options, maxName); err != nil {
		return err
	}
	if err := validateKeys(s.Components.Keys); err != nil {
		return err
	}
	if err := validateProtocols(s.Components.Protocols); err != nil {
		return err
	}
	if err := validateRPCs(s.Components.RPCs, maxName); err != nil {
		return err
	}
	if err := validateMailGroups(s.Components.MailGroups); err != nil {
		return err
	}
	if err := validateListTemplates(s.Components.ListTemplates); err != nil {
		return err
	}
	if err := validateHelpFrames(s.Components.HelpFrames); err != nil {
		return err
	}
	if err := validateHL7Applications(s.Components.HL7Applications); err != nil {
		return err
	}
	if err := validateHLOApplications(s.Components.HLOApplications); err != nil {
		return err
	}
	if err := validateLogicalLinks(s.Components.LogicalLinks); err != nil {
		return err
	}
	if err := validateFiles(s.Components.Files); err != nil {
		return err
	}
	for _, rb := range s.RequiredBuilds {
		if !reReqBuild.MatchString(rb.Name) {
			return fmt.Errorf("buildspec: required build name %q must be NAMESPACE*VERSION[*PATCH]", rb.Name)
		}
		if !validActions[rb.Action] {
			return fmt.Errorf("buildspec: required build %s action %q is not a KIDS install action", rb.Name, rb.Action)
		}
	}
	if err := s.validateInstallHooks(maxName); err != nil {
		return err
	}
	for _, icr := range s.ICRs {
		if icr.Number <= 0 {
			return fmt.Errorf("buildspec: ICR number must be positive, got %d", icr.Number)
		}
	}
	return nil
}

// validateInstallHooks checks the env-check / pre-install / post-install routine
// declarations (B.3). Env-check is a bare routine name (ENV does D @("^"_name), so
// no tag); pre/post are entryrefs (TAG^RTN or RTN) that PRE^/POST^XPDIJ1 D @. Only
// the shape is validated — the routine may be shipped in this build or pre-exist on
// the target (a missing one surfaces clearly at install time).
func (s *Spec) validateInstallHooks(maxName int) error {
	if s.EnvCheck != "" && !isRoutineName(s.EnvCheck, maxName) {
		return fmt.Errorf("buildspec: envCheck %q must be a bare routine name (≤%d chars, no tag)", s.EnvCheck, maxName)
	}
	for _, h := range []struct{ field, ref string }{
		{"preInstall", s.PreInstall},
		{"postInstall", s.PostInstall},
	} {
		if h.ref == "" {
			continue
		}
		tag, rtn, ok := splitEntryref(h.ref)
		if !ok || !isRoutineName(rtn, maxName) || (tag != "" && !reLabel.MatchString(tag)) {
			return fmt.Errorf("buildspec: %s %q must be a routine entryref (ROUTINE or TAG^ROUTINE)", h.field, h.ref)
		}
	}
	return nil
}

// splitEntryref splits an M entryref into its optional tag and routine. "RTN" →
// ("", "RTN"); "TAG^RTN" → ("TAG", "RTN"). ok is false on an empty part or extra
// carets.
func splitEntryref(s string) (tag, rtn string, ok bool) {
	switch parts := strings.Split(s, "^"); len(parts) {
	case 1:
		return "", parts[0], parts[0] != ""
	case 2:
		return parts[0], parts[1], parts[0] != "" && parts[1] != ""
	default:
		return "", "", false
	}
}

// validateParamDefs checks each PARAMETER DEFINITION component: a valid #8989.51
// NAME (≤30 chars, uppercase), a known value data type, and known allowable
// entities with non-negative precedence.
func validateParamDefs(defs []ParamDef) error {
	for _, p := range defs {
		if p.Name == "" {
			return fmt.Errorf("buildspec: a parameterDefinition is missing its name")
		}
		if len(p.Name) > 30 || !reParamName.MatchString(p.Name) {
			return fmt.Errorf("buildspec: parameter name %q must be uppercase ≤30 chars (#8989.51 NAME)", p.Name)
		}
		if p.DataType != "" {
			if _, ok := ParamDataTypeCode[p.DataType]; !ok {
				return fmt.Errorf("buildspec: parameter %s data type %q is not a known #8989.51 value type", p.Name, p.DataType)
			}
		}
		for _, e := range p.Entities {
			if _, ok := ParamEntityIEN[e.Entity]; !ok {
				return fmt.Errorf("buildspec: parameter %s allowable entity %q is not a known parameter entity (SYS, USR, …)", p.Name, e.Entity)
			}
			if e.Precedence < 0 {
				return fmt.Errorf("buildspec: parameter %s entity %s precedence must be ≥ 0", p.Name, e.Entity)
			}
		}
	}
	return nil
}

// validateOptions checks each OPTION component: a valid #19 NAME (≤30 chars,
// uppercase), a known option type, and a well-formed ROUTINE entryref when set
// (required for a "run routine" type). The option's record is filed by KRN^XPDIK;
// only its build-side shape is validated here.
func validateOptions(opts []OptionComp, maxName int) error {
	for _, o := range opts {
		if o.Name == "" {
			return fmt.Errorf("buildspec: an option is missing its name")
		}
		if len(o.Name) > 30 || !reEntryName.MatchString(o.Name) {
			return fmt.Errorf("buildspec: option name %q must be uppercase ≤30 chars (#19 OPTION NAME)", o.Name)
		}
		if _, ok := OptionTypeCode[o.Type]; !ok {
			return fmt.Errorf("buildspec: option %s type %q is not a known option type (menu, run routine, action, …)", o.Name, o.Type)
		}
		if o.Type == "run routine" && o.Routine == "" {
			return fmt.Errorf("buildspec: option %s is a run-routine option but has no routine entryref", o.Name)
		}
		if o.Routine != "" {
			tag, rtn, ok := splitEntryref(o.Routine)
			if !ok || !isRoutineName(rtn, maxName) || (tag != "" && !reLabel.MatchString(tag)) {
				return fmt.Errorf("buildspec: option %s routine %q must be a routine entryref (ROUTINE or TAG^ROUTINE)", o.Name, o.Routine)
			}
		}
	}
	return nil
}

// validateRPCs checks each REMOTE PROCEDURE component: a valid #8994 NAME (≤30
// chars, uppercase), a ROUTINE that is a valid M routine name (the DD-required
// .03), an optional TAG that is a valid M label, and a known return-value type (or
// empty, defaulted to "single value" at emit). The record is filed by KRN^XPDIK.
func validateRPCs(rpcs []RPCComp, maxName int) error {
	for _, r := range rpcs {
		if r.Name == "" {
			return fmt.Errorf("buildspec: an rpc is missing its name")
		}
		if len(r.Name) > 30 || !reEntryName.MatchString(r.Name) {
			return fmt.Errorf("buildspec: rpc name %q must be uppercase ≤30 chars (#8994 REMOTE PROCEDURE NAME)", r.Name)
		}
		if !isRoutineName(r.Routine, maxName) {
			return fmt.Errorf("buildspec: rpc %s routine %q must be a valid M routine name (#8994 ROUTINE is required)", r.Name, r.Routine)
		}
		if r.Tag != "" && !reLabel.MatchString(r.Tag) {
			return fmt.Errorf("buildspec: rpc %s tag %q must be a valid M label", r.Name, r.Tag)
		}
		if r.ReturnType != "" {
			if _, ok := RPCReturnTypeCode[r.ReturnType]; !ok {
				return fmt.Errorf("buildspec: rpc %s returnType %q is not a known #8994 return-value type", r.Name, r.ReturnType)
			}
		}
		for _, p := range r.InputParameters {
			if !reEntryName.MatchString(p.Name) || len(p.Name) > 30 {
				return fmt.Errorf("buildspec: rpc %s input parameter %q must be uppercase ≤30 chars (#8994.02 INPUT PARAMETER)", r.Name, p.Name)
			}
			if p.Type != "" {
				if _, ok := RPCParamTypeCode[p.Type]; !ok {
					return fmt.Errorf("buildspec: rpc %s parameter %s type %q is not a known #8994.02 PARAMETER TYPE", r.Name, p.Name, p.Type)
				}
			}
			if p.MaxLength < 0 || p.Sequence < 0 {
				return fmt.Errorf("buildspec: rpc %s parameter %s maxLength/sequence must be non-negative", r.Name, p.Name)
			}
		}
	}
	return nil
}

// validateProtocols checks each PROTOCOL component: a valid #101 NAME (≤30 chars,
// uppercase) and a known protocol type. The record is filed by KRN^XPDIK; only its
// build-side shape is validated here.
func validateProtocols(protos []ProtocolComp) error {
	for _, p := range protos {
		if p.Name == "" {
			return fmt.Errorf("buildspec: a protocol is missing its name")
		}
		if len(p.Name) > 30 || !reEntryName.MatchString(p.Name) {
			return fmt.Errorf("buildspec: protocol name %q must be uppercase ≤30 chars (#101 PROTOCOL NAME)", p.Name)
		}
		if _, ok := ProtocolTypeCode[p.Type]; !ok {
			return fmt.Errorf("buildspec: protocol %s type %q is not a known protocol type (action, extended action, menu, …)", p.Name, p.Type)
		}
		for _, it := range p.Items {
			if len(it.Name) > 30 || !reEntryName.MatchString(it.Name) {
				return fmt.Errorf("buildspec: protocol %s item %q must be an uppercase ≤30-char #101 PROTOCOL name", p.Name, it.Name)
			}
			if it.Sequence < 0 {
				return fmt.Errorf("buildspec: protocol %s item %s sequence must be non-negative", p.Name, it.Name)
			}
		}
	}
	return nil
}

// validateKeys checks each SECURITY KEY component: a valid #19.1 NAME (≤30 chars,
// uppercase). The key record is filed by KRN^XPDIK; only its build-side shape is
// validated here.
func validateKeys(keys []KeyComp) error {
	for _, k := range keys {
		if k.Name == "" {
			return fmt.Errorf("buildspec: a key is missing its name")
		}
		if len(k.Name) > 30 || !reEntryName.MatchString(k.Name) {
			return fmt.Errorf("buildspec: key name %q must be uppercase ≤30 chars (#19.1 SECURITY KEY NAME)", k.Name)
		}
	}
	return nil
}

// validateMailGroups checks each MAIL GROUP component: a valid #3.8 NAME (≤30
// chars, uppercase) and, when given, a known type (public/private). The record is
// filed by KRN^XPDIK; only its build-side shape is validated here.
func validateMailGroups(mgs []MailGroupComp) error {
	for _, m := range mgs {
		if m.Name == "" {
			return fmt.Errorf("buildspec: a mail group is missing its name")
		}
		if len(m.Name) > 30 || !reEntryName.MatchString(m.Name) {
			return fmt.Errorf("buildspec: mail group name %q must be uppercase ≤30 chars (#3.8 MAIL GROUP NAME)", m.Name)
		}
		if m.Type != "" {
			if _, ok := MailGroupTypeCode[m.Type]; !ok {
				return fmt.Errorf("buildspec: mail group %s type %q is not a known type (public, private)", m.Name, m.Type)
			}
		}
	}
	return nil
}

// validateListTemplates checks each LIST TEMPLATE component: a valid #409.61 NAME
// (≤30 chars, uppercase) and screen margins within an 0–255 terminal range. The
// record is filed by KRN^XPDIK; only its build-side shape is validated here.
func validateListTemplates(lts []ListTemplateComp) error {
	for _, lt := range lts {
		if lt.Name == "" {
			return fmt.Errorf("buildspec: a list template is missing its name")
		}
		if len(lt.Name) > 30 || !reEntryName.MatchString(lt.Name) {
			return fmt.Errorf("buildspec: list template name %q must be uppercase ≤30 chars (#409.61 LIST TEMPLATE NAME)", lt.Name)
		}
		for _, m := range []struct {
			label string
			v     int
		}{{"rightMargin", lt.RightMargin}, {"topMargin", lt.TopMargin}, {"bottomMargin", lt.BottomMargin}} {
			if m.v < 0 || m.v > 255 {
				return fmt.Errorf("buildspec: list template %s %s %d out of range (0–255)", lt.Name, m.label, m.v)
			}
		}
	}
	return nil
}

// validateHelpFrames checks each HELP FRAME component: a valid #9.2 NAME (3–30
// chars, uppercase, spaces/hyphens, not all-digit — matching the live .01 input
// transform). The TEXT body is shipped verbatim. The record is filed by KRN^XPDIK;
// only its build-side shape is validated here.
func validateHelpFrames(hfs []HelpFrameComp) error {
	for _, h := range hfs {
		if h.Name == "" {
			return fmt.Errorf("buildspec: a help frame is missing its name")
		}
		if l := len(h.Name); l < 3 || l > 30 || !reHelpName.MatchString(h.Name) {
			return fmt.Errorf("buildspec: help frame name %q must be uppercase 3–30 chars (spaces/hyphens ok) (#9.2 HELP FRAME NAME)", h.Name)
		}
	}
	return nil
}

// validateHL7Applications checks each HL7 APPLICATION PARAMETER component: a valid
// #771 NAME (≤30 chars, uppercase, spaces/underscores). The record is filed by
// KRN^XPDIK; only its build-side shape is validated here.
func validateHL7Applications(apps []HL7AppComp) error {
	for _, a := range apps {
		if a.Name == "" {
			return fmt.Errorf("buildspec: an hl7 application is missing its name")
		}
		if len(a.Name) > 30 || !reHL7Name.MatchString(a.Name) {
			return fmt.Errorf("buildspec: hl7 application name %q must be uppercase ≤30 chars (spaces/underscores ok) (#771 HL7 APPLICATION PARAMETER NAME)", a.Name)
		}
	}
	return nil
}

// validateHLOApplications checks each HLO APPLICATION REGISTRY component: a valid
// #779.2 APPLICATION NAME (3–60 chars, uppercase, spaces/underscores) and, for each
// MESSAGE TYPE ACTIONS entry, an uppercase HL7 message type + event code and (when
// present) a valid action routine. The records are filed by KRN^XPDIK; only their
// build-side shape is validated here.
func validateHLOApplications(apps []HLOAppComp) error {
	for _, a := range apps {
		if a.Name == "" {
			return fmt.Errorf("buildspec: an hlo application is missing its name")
		}
		if n := len(a.Name); n < 3 || n > 60 || !reHL7Name.MatchString(a.Name) {
			return fmt.Errorf("buildspec: hlo application name %q must be uppercase 3–60 chars (spaces/underscores ok) (#779.2 HLO APPLICATION REGISTRY NAME)", a.Name)
		}
		for _, mt := range a.MessageTypes {
			if !reHL7Code.MatchString(mt.MessageType) {
				return fmt.Errorf("buildspec: hlo application %q message type %q must be an uppercase HL7 code (#779.21 HL7 MESSAGE TYPE)", a.Name, mt.MessageType)
			}
			if !reHL7Code.MatchString(mt.Event) {
				return fmt.Errorf("buildspec: hlo application %q event %q must be an uppercase HL7 code (#779.21 HL7 EVENT)", a.Name, mt.Event)
			}
			if mt.ActionRoutine != "" && !reRoutine.MatchString(mt.ActionRoutine) {
				return fmt.Errorf("buildspec: hlo application %q action routine %q is not a valid M routine name (#779.21 ACTION ROUTINE)", a.Name, mt.ActionRoutine)
			}
		}
	}
	return nil
}

// validateLogicalLinks checks each HL LOGICAL LINK component: a valid #870 NODE
// name (3–10 chars, no leading punctuation — matching the live .01 input transform)
// and, when present, a valid SERVICE TYPE set code (C/S/M). LLP TYPE and the TCP
// params are shipped verbatim and resolved/filed by KRN^XPDIK.
func validateLogicalLinks(lls []LogicalLinkComp) error {
	for _, l := range lls {
		if l.Name == "" {
			return fmt.Errorf("buildspec: a logical link is missing its name")
		}
		if n := len(l.Name); n < 3 || n > 10 || !reLinkName.MatchString(l.Name) {
			return fmt.Errorf("buildspec: logical link name %q must be 3–10 chars, no leading punctuation (#870 HL LOGICAL LINK NODE)", l.Name)
		}
		if l.ServiceType != "" && !reSvcType.MatchString(l.ServiceType) {
			return fmt.Errorf("buildspec: logical link %q service type %q must be C, S, or M (#870 400.03 TCP/IP SERVICE TYPE)", l.Name, l.ServiceType)
		}
	}
	return nil
}

// validateFiles checks each FILE component is a brand-new throwaway file: a
// number in the local/test range, a valid uppercase FileMan name, and a
// well-formed data global root (defaulted to ^DIZ(<file>, when omitted). It
// mutates each FileComp in place to fill the default global root.
func validateFiles(files []FileComp) error {
	for i := range files {
		f := &files[i]
		n := int64(f.Number)
		if float64(n) != f.Number || n < LocalFileMin || n > LocalFileMax {
			return fmt.Errorf("buildspec: file number %v must be an integer in the local range %d-%d", f.Number, LocalFileMin, LocalFileMax)
		}
		if f.Name == "" {
			return fmt.Errorf("buildspec: file #%d is missing its name", n)
		}
		if len(f.Name) > 30 || !reFileName.MatchString(f.Name) {
			return fmt.Errorf("buildspec: file name %q must be uppercase ≤30 chars (FileMan FILE name)", f.Name)
		}
		if f.GlobalRoot == "" {
			f.GlobalRoot = fmt.Sprintf("^DIZ(%d,", n)
		}
		if !reGlobalRt.MatchString(f.GlobalRoot) {
			return fmt.Errorf("buildspec: file %s global root %q must be an open global ref ending in a comma, e.g. ^DIZ(%d,", f.Name, f.GlobalRoot, n)
		}
		if err := validateFields(n, f.Fields); err != nil {
			return err
		}
	}
	return nil
}

// validateFields checks each field beyond the .01 NAME: a field number above .01
// (unique within the file), a valid uppercase label, a known type, a non-colliding
// storage location ((0;1) is reserved for the .01), and type-specific knobs.
func validateFields(fileNum int64, fields []FieldSpec) error {
	usedLoc := map[string]bool{"0;1": true} // the .01 occupies node 0, piece 1
	usedNum := map[float64]bool{0.01: true}
	for i := range fields {
		f := &fields[i]
		if f.Number <= 0.01 {
			return fmt.Errorf("buildspec: file #%d field number %v must be greater than .01 (the NAME field)", fileNum, f.Number)
		}
		if usedNum[f.Number] {
			return fmt.Errorf("buildspec: file #%d has a duplicate field number %v", fileNum, f.Number)
		}
		usedNum[f.Number] = true
		if f.Label == "" || len(f.Label) > 30 || !reFileName.MatchString(f.Label) {
			return fmt.Errorf("buildspec: file #%d field %v label %q must be uppercase ≤30 chars", fileNum, f.Number, f.Label)
		}
		if !validFieldTypes[f.Type] {
			return fmt.Errorf("buildspec: file #%d field %s type %q is not a known field type (free text, numeric, date, set of codes, pointer)", fileNum, f.Label, f.Type)
		}
		if f.Node < 0 {
			return fmt.Errorf("buildspec: file #%d field %s storage node %d must be ≥ 0", fileNum, f.Label, f.Node)
		}
		if f.Piece < 1 {
			return fmt.Errorf("buildspec: file #%d field %s must specify a storage piece ≥ 1", fileNum, f.Label)
		}
		loc := fmt.Sprintf("%d;%d", f.Node, f.Piece)
		if usedLoc[loc] {
			return fmt.Errorf("buildspec: file #%d field %s storage %s collides with another field or the .01 NAME", fileNum, f.Label, loc)
		}
		usedLoc[loc] = true
		if err := validateFieldType(fileNum, f); err != nil {
			return err
		}
	}
	return nil
}

// validateFieldType checks the type-specific knobs of one field.
func validateFieldType(fileNum int64, f *FieldSpec) error {
	switch f.Type {
	case "numeric":
		if f.Width < 1 {
			return fmt.Errorf("buildspec: file #%d numeric field %s needs a positive width", fileNum, f.Label)
		}
		if f.Decimals < 0 {
			return fmt.Errorf("buildspec: file #%d numeric field %s decimals %d must be ≥ 0", fileNum, f.Label, f.Decimals)
		}
		if f.Min != nil && f.Max != nil && *f.Min > *f.Max {
			return fmt.Errorf("buildspec: file #%d numeric field %s min %v is above max %v", fileNum, f.Label, *f.Min, *f.Max)
		}
	case "set of codes":
		if len(f.Codes) == 0 {
			return fmt.Errorf("buildspec: file #%d set-of-codes field %s needs at least one code", fileNum, f.Label)
		}
		for _, c := range f.Codes {
			if c.Internal == "" || c.External == "" {
				return fmt.Errorf("buildspec: file #%d field %s has a code with an empty internal or external value", fileNum, f.Label)
			}
			if strings.ContainsAny(c.Internal+c.External, ":;^") {
				return fmt.Errorf("buildspec: file #%d field %s code %q/%q must not contain ':', ';' or '^'", fileNum, f.Label, c.Internal, c.External)
			}
		}
	case "pointer":
		if f.PointTo <= 0 {
			return fmt.Errorf("buildspec: file #%d pointer field %s needs a positive pointTo file number", fileNum, f.Label)
		}
		if !reGlobalRt.MatchString(f.PointRoot) {
			return fmt.Errorf("buildspec: file #%d pointer field %s pointRoot %q must be an open global ref ending in a comma", fileNum, f.Label, f.PointRoot)
		}
	case "free text":
		if f.MaxLen < 0 || f.MaxLen > freeTextMaxLen {
			return fmt.Errorf("buildspec: file #%d free-text field %s maxLen %d must be 0-%d", fileNum, f.Label, f.MaxLen, freeTextMaxLen)
		}
	}
	return nil
}

func validateRoutines(rtns []string, maxLen int) error {
	for _, r := range rtns {
		if !isRoutineName(r, maxLen) {
			return fmt.Errorf("buildspec: %q is not a valid routine name (≤%d chars, M naming)", r, maxLen)
		}
	}
	return nil
}

// isRoutineName reports whether s is a valid VistA/M routine name: M-naming
// characters (optional leading %, then uppercase letters/digits) and length in
// [1, maxLen]. maxLen is RoutineNameMaxStd by default, or RoutineNameMaxLong
// when the spec sets allowLongNames.
func isRoutineName(s string, maxLen int) bool {
	return len(s) >= 1 && len(s) <= maxLen && reRoutine.MatchString(s) && !strings.ContainsAny(s, " \t")
}
