package pkgcli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vista-cloud-dev/clikit"
	"github.com/vista-cloud-dev/v-pkg/internal/buildspec"
	"github.com/vista-cloud-dev/v-pkg/internal/kids"
)

// buildCmd is `v pkg build` (VSL T0a.2): assemble a KIDS transport global from a
// declarative build spec (kids/<pkg>.build.json) + the routine source, producing
// a NORMALIZED, byte-identical export — the deterministic-build invariant
// (coordination plan §7.2 #2). Unlike `assemble` (which reassembles an existing
// decomposed .KID tree), `build` constructs the package from its git source of
// truth.
type buildCmd struct {
	Spec string `arg:"" help:"Path to the kids/<pkg>.build.json build spec."`
	Src  string `help:"Directory holding the routine source (<routine>.m)." default:"src" placeholder:"DIR"`
	Out  string `help:"Output .KID path (default: dist/kids/<pkg>.kids)." placeholder:"PATH"`
}

type buildResult struct {
	InstallName     string `json:"installName"`
	Out             string `json:"out"`
	Routines        int    `json:"routines"`
	ParamDefs       int    `json:"paramDefs,omitempty"`
	Options         int    `json:"options,omitempty"`
	Keys            int    `json:"keys,omitempty"`
	Protocols       int    `json:"protocols,omitempty"`
	RPCs            int    `json:"rpcs,omitempty"`
	MailGroups      int    `json:"mailGroups,omitempty"`
	ListTemplates   int    `json:"listTemplates,omitempty"`
	HelpFrames      int    `json:"helpFrames,omitempty"`
	HL7Applications int    `json:"hl7Applications,omitempty"`
	LogicalLinks    int    `json:"logicalLinks,omitempty"`
	Files           int    `json:"files,omitempty"`
	RequiredBuilds  int    `json:"requiredBuilds,omitempty"`
}

func (c *buildCmd) Run(cc *clikit.Context) error {
	spec, err := buildspec.Load(c.Spec)
	if err != nil {
		return clikit.Fail(clikit.ExitUsage, "BAD_SPEC", err.Error(), "fix the build spec")
	}

	rtns := make([]kids.RoutineSrc, 0, len(spec.Components.Routines))
	for _, name := range spec.Components.Routines {
		p := filepath.Join(c.Src, name+".m")
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return clikit.Fail(clikit.ExitRuntime, "READ_FAILED", rerr.Error(),
				"stage the routine source under --src (e.g. "+filepath.Join(c.Src, name+".m")+")")
		}
		rtns = append(rtns, kids.RoutineSrc{Name: name, Lines: routineLines(data)})
	}

	paramDefs, perr := resolveParamDefs(spec.Components.ParameterDefinitions)
	if perr != nil {
		return clikit.Fail(clikit.ExitUsage, "BAD_SPEC", perr.Error(), "fix the parameterDefinitions in the build spec")
	}
	files := resolveFiles(spec.Components.Files)
	options := resolveOptions(spec.Components.Options)
	keys := resolveKeys(spec.Components.Keys)
	protocols := resolveProtocols(spec.Components.Protocols)
	rpcs := resolveRPCs(spec.Components.RPCs)
	mailGroups := resolveMailGroups(spec.Components.MailGroups)
	listTemplates := resolveListTemplates(spec.Components.ListTemplates)
	helpFrames := resolveHelpFrames(spec.Components.HelpFrames)
	hl7Apps := resolveHL7Apps(spec.Components.HL7Applications)
	logicalLinks := resolveLogicalLinks(spec.Components.LogicalLinks)
	reqBuilds := resolveRequiredBuilds(spec.RequiredBuilds)

	pairs := kids.MakeBuildPairs(kids.BuildInput{
		InstallName:    spec.InstallName(),
		Namespace:      spec.Package,
		Routines:       rtns,
		ParamDefs:      paramDefs,
		Options:        options,
		Keys:           keys,
		Protocols:      protocols,
		RPCs:           rpcs,
		MailGroups:     mailGroups,
		ListTemplates:  listTemplates,
		HelpFrames:     helpFrames,
		HL7Apps:        hl7Apps,
		LogicalLinks:   logicalLinks,
		Files:          files,
		RequiredBuilds: reqBuilds,
		EnvCheck:       spec.EnvCheck,
		PreInstall:     spec.PreInstall,
		PostInstall:    spec.PostInstall,
	})

	out := c.Out
	if out == "" {
		out = filepath.Join("dist", "kids", spec.Package+".kids")
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return clikit.Fail(clikit.ExitRuntime, "WRITE_FAILED", err.Error(), "")
	}
	if err := kids.WriteKID([]string{spec.InstallName()},
		map[string][]kids.Pair{spec.InstallName(): pairs}, out); err != nil {
		return clikit.Fail(clikit.ExitRuntime, "WRITE_FAILED", err.Error(), "")
	}

	return cc.Result(buildResult{
		InstallName: spec.InstallName(), Out: out, Routines: len(rtns),
		ParamDefs: len(paramDefs), Options: len(options), Keys: len(keys), Protocols: len(protocols), RPCs: len(rpcs), MailGroups: len(mailGroups), ListTemplates: len(listTemplates), HelpFrames: len(helpFrames), HL7Applications: len(hl7Apps), LogicalLinks: len(logicalLinks), Files: len(files), RequiredBuilds: len(reqBuilds),
	}, func() {
		cc.Title("pkg build")
		fmt.Fprintf(cc.Stdout, "%s built %s (%d routine(s), %d param def(s), %d option(s), %d key(s), %d protocol(s), %d rpc(s), %d mail group(s), %d list template(s), %d help frame(s), %d hl7 app(s), %d logical link(s), %d file(s), %d required build(s)) → %s\n",
			cc.Success("ok"), cc.Accent(spec.InstallName()), len(rtns), len(paramDefs), len(options), len(keys), len(protocols), len(rpcs), len(mailGroups), len(listTemplates), len(helpFrames), len(hl7Apps), len(logicalLinks), len(files), len(reqBuilds), cc.Accent(out))
	})
}

// resolveParamDefs maps the human PARAMETER DEFINITION spec onto the kids emit
// shape — value-data-type name → #8989.51 code, entity abbreviation → #8989.518
// IEN. The spec is already validated, so the lookups are guaranteed present;
// the error guards a future spec that slips an unknown key past validation.
func resolveParamDefs(defs []buildspec.ParamDef) ([]kids.ParamDef, error) {
	out := make([]kids.ParamDef, 0, len(defs))
	for _, d := range defs {
		dtName := d.DataType
		if dtName == "" {
			dtName = "free text"
		}
		dt, ok := buildspec.ParamDataTypeCode[dtName]
		if !ok {
			return nil, fmt.Errorf("parameter %s: unknown data type %q", d.Name, d.DataType)
		}
		ents := make([]kids.ParamEntity, 0, len(d.Entities))
		for _, e := range d.Entities {
			ien, ok := buildspec.ParamEntityIEN[e.Entity]
			if !ok {
				return nil, fmt.Errorf("parameter %s: unknown entity %q", d.Name, e.Entity)
			}
			ents = append(ents, kids.ParamEntity{EntityIEN: ien, Precedence: e.Precedence})
		}
		out = append(out, kids.ParamDef{
			Name: d.Name, DisplayText: d.DisplayText, DataTypeCode: dt, Entities: ents,
		})
	}
	return out, nil
}

// resolveFiles maps the spec's FILE components onto the kids emit shape. The spec
// is already validated (number in range, name valid, global root defaulted), so
// the FileMan file number fits an int64.
func resolveFiles(files []buildspec.FileComp) []kids.FileDD {
	out := make([]kids.FileDD, 0, len(files))
	for _, f := range files {
		out = append(out, kids.FileDD{
			Number:     int64(f.Number),
			Name:       f.Name,
			GlobalRoot: f.GlobalRoot,
			Fields:     resolveFields(f.Fields),
		})
	}
	return out
}

// resolveFields maps a spec's field declarations onto the kids emit shape. The
// spec is already validated (type known, storage non-colliding, type-specific
// knobs present), so the values carry straight through.
func resolveFields(fields []buildspec.FieldSpec) []kids.FileField {
	if len(fields) == 0 {
		return nil
	}
	out := make([]kids.FileField, 0, len(fields))
	for _, f := range fields {
		ff := kids.FileField{
			Number:    f.Number,
			Label:     f.Label,
			Type:      f.Type,
			Node:      f.Node,
			Piece:     f.Piece,
			Required:  f.Required,
			Help:      f.Help,
			MaxLen:    f.MaxLen,
			Width:     f.Width,
			Decimals:  f.Decimals,
			Min:       f.Min,
			Max:       f.Max,
			HasTime:   f.Time,
			PointTo:   f.PointTo,
			PointRoot: f.PointRoot,
		}
		for _, c := range f.Codes {
			ff.Codes = append(ff.Codes, kids.SetCode{Internal: c.Internal, External: c.External})
		}
		out = append(out, ff)
	}
	return out
}

// resolveOptions maps the spec's OPTION components onto the kids emit shape —
// option-type name → #19 field 4 (TYPE) set-of-codes value. The spec is already
// validated (type known, run-routine has a routine), so the lookup is guaranteed.
func resolveOptions(opts []buildspec.OptionComp) []kids.Option {
	out := make([]kids.Option, 0, len(opts))
	for _, o := range opts {
		out = append(out, kids.Option{
			Name:        o.Name,
			MenuText:    o.MenuText,
			TypeCode:    buildspec.OptionTypeCode[o.Type],
			Routine:     o.Routine,
			EntryAction: o.EntryAction,
			ExitAction:  o.ExitAction,
		})
	}
	return out
}

// resolveKeys maps the spec's SECURITY KEY components onto the kids emit shape.
func resolveKeys(keys []buildspec.KeyComp) []kids.SecurityKey {
	out := make([]kids.SecurityKey, 0, len(keys))
	for _, k := range keys {
		out = append(out, kids.SecurityKey{Name: k.Name})
	}
	return out
}

// resolveProtocols maps the spec's PROTOCOL components onto the kids emit shape —
// protocol-type name → #101 field 4 (TYPE) set-of-codes value. The spec is already
// validated (type known), so the lookup is guaranteed present.
func resolveProtocols(protos []buildspec.ProtocolComp) []kids.Protocol {
	out := make([]kids.Protocol, 0, len(protos))
	for _, p := range protos {
		out = append(out, kids.Protocol{
			Name:        p.Name,
			ItemText:    p.ItemText,
			TypeCode:    buildspec.ProtocolTypeCode[p.Type],
			EntryAction: p.EntryAction,
			ExitAction:  p.ExitAction,
		})
	}
	return out
}

// resolveRPCs maps the spec's REMOTE PROCEDURE components onto the kids emit shape
// — return-value-type name → #8994 field .04 set-of-codes value, defaulting to
// "single value" (code 1) when omitted (the field is DD-required).
func resolveRPCs(rpcs []buildspec.RPCComp) []kids.RPC {
	out := make([]kids.RPC, 0, len(rpcs))
	for _, r := range rpcs {
		rt := r.ReturnType
		if rt == "" {
			rt = "single value"
		}
		out = append(out, kids.RPC{
			Name:           r.Name,
			Tag:            r.Tag,
			Routine:        r.Routine,
			ReturnTypeCode: buildspec.RPCReturnTypeCode[rt],
		})
	}
	return out
}

// resolveMailGroups maps the spec's MAIL GROUP components onto the kids emit shape
// — type name → #3.8 field 4 set-of-codes value (default "public" → "PU", since the
// field is DD-required), and the self-enrollment bool → the field 7 y/n code.
func resolveMailGroups(mgs []buildspec.MailGroupComp) []kids.MailGroup {
	out := make([]kids.MailGroup, 0, len(mgs))
	for _, m := range mgs {
		typ := m.Type
		if typ == "" {
			typ = "public"
		}
		self := ""
		if m.AllowSelfEnrollment {
			self = "y"
		}
		out = append(out, kids.MailGroup{
			Name:            m.Name,
			TypeCode:        buildspec.MailGroupTypeCode[typ],
			AllowSelfEnroll: self,
		})
	}
	return out
}

// resolveListTemplates maps the spec's LIST TEMPLATE components onto the kids emit
// shape, applying the screen-geometry defaults (right 80, top 3, bottom 20) when a
// margin is omitted — a List Manager screen needs a bounded region to render in.
func resolveListTemplates(lts []buildspec.ListTemplateComp) []kids.ListTemplate {
	dflt := func(v, d int) string {
		if v == 0 {
			v = d
		}
		return strconv.Itoa(v)
	}
	out := make([]kids.ListTemplate, 0, len(lts))
	for _, lt := range lts {
		out = append(out, kids.ListTemplate{
			Name:         lt.Name,
			ScreenTitle:  lt.ScreenTitle,
			ProtocolMenu: lt.ProtocolMenu,
			RightMargin:  dflt(lt.RightMargin, 80),
			TopMargin:    dflt(lt.TopMargin, 3),
			BottomMargin: dflt(lt.BottomMargin, 20),
			HeaderCode:   lt.HeaderCode,
			EntryCode:    lt.EntryCode,
			ExitCode:     lt.ExitCode,
			HelpCode:     lt.HelpCode,
			ArrayName:    lt.ArrayName,
		})
	}
	return out
}

// resolveHelpFrames maps the spec's HELP FRAME components onto the kids emit shape
// — a straight field carry-through (the volatile dates are dropped at emit time).
func resolveHelpFrames(hfs []buildspec.HelpFrameComp) []kids.HelpFrame {
	out := make([]kids.HelpFrame, 0, len(hfs))
	for _, h := range hfs {
		out = append(out, kids.HelpFrame{Name: h.Name, Header: h.Header, Text: h.Text})
	}
	return out
}

// resolveHL7Apps maps the spec's HL7 APPLICATION PARAMETER components onto the kids
// emit shape, defaulting the COUNTRY CODE to "USA" (the universal shipped value)
// when omitted.
func resolveHL7Apps(apps []buildspec.HL7AppComp) []kids.HL7App {
	out := make([]kids.HL7App, 0, len(apps))
	for _, a := range apps {
		cc := a.CountryCode
		if cc == "" {
			cc = "USA"
		}
		out = append(out, kids.HL7App{Name: a.Name, Facility: a.Facility, CountryCode: cc})
	}
	return out
}

// resolveLogicalLinks maps the spec's HL LOGICAL LINK components onto the kids
// emit shape, defaulting the LLP TYPE to "TCP" (resolved to its #869.1 IEN at
// install) and, when a PORT is present, the SERVICE TYPE to "C" (CLIENT/SENDER) —
// the common case for an outbound link.
func resolveLogicalLinks(lls []buildspec.LogicalLinkComp) []kids.LogicalLink {
	out := make([]kids.LogicalLink, 0, len(lls))
	for _, l := range lls {
		llp := l.LLPType
		if llp == "" {
			llp = "TCP"
		}
		svc := l.ServiceType
		if svc == "" && l.Port != "" {
			svc = "C"
		}
		out = append(out, kids.LogicalLink{
			Name: l.Name, LLPType: llp, Port: l.Port, ServiceType: svc,
		})
	}
	return out
}

// resolveRequiredBuilds maps the spec's Required Builds onto the kids emit shape,
// turning the action phrase into its #9.611 ACTION code.
func resolveRequiredBuilds(reqs []buildspec.RequiredBuild) []kids.ReqBuild {
	out := make([]kids.ReqBuild, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, kids.ReqBuild{Name: r.Name, Action: buildspec.RequiredBuildActionCode[r.Action]})
	}
	return out
}

// routineLines splits routine source into lines, dropping a single trailing
// newline (so a normal text file does not yield a spurious empty final line).
func routineLines(data []byte) []string {
	s := strings.TrimRight(string(data), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
