package clusterapply

import (
	"fmt"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	uitable "github.com/cppforlife/go-cli-ui/ui/table"
	cmdcore "github.com/k14s/kapp/pkg/kapp/cmd/core"
	ctldiff "github.com/k14s/kapp/pkg/kapp/diff"
	ctlres "github.com/k14s/kapp/pkg/kapp/resources"
)

type ChangeView interface {
	Resource() ctlres.Resource
	ApplyOp() ClusterChangeApplyOp
	WaitOp() ClusterChangeWaitOp
	TextDiff() ctldiff.TextDiff
}

type ChangesView struct {
	ChangeViews []ChangeView
	Sort        bool

	countsView *ChangesCountsView
}

func (v *ChangesView) Print(ui ui.UI) {
	versionHeader := uitable.NewHeader("Version")
	versionHeader.Hidden = true

	conditionsHeader := uitable.NewHeader("Conditions")
	conditionsHeader.Title = "Conds."

	table := uitable.Table{
		Title: "Changes",
		// TODO do not show total number of "changes" as it may
		// be confusing that some changes are only waits
		// Content: "changes",

		Header: []uitable.Header{
			uitable.NewHeader("Namespace"),
			uitable.NewHeader("Name"),
			uitable.NewHeader("Kind"),
			versionHeader,
			conditionsHeader,
			uitable.NewHeader("Age"),
			uitable.NewHeader("Op"),
			uitable.NewHeader("Wait to"),
		},
	}

	if v.Sort {
		table.SortBy = []uitable.ColumnSort{
			{Column: 0, Asc: true},
			{Column: 1, Asc: true},
			{Column: 2, Asc: true},
			{Column: 3, Asc: true},
		}
	} else {
		// Otherwise it might look very awkward
		table.FillFirstColumn = true
	}

	v.countsView = NewChangesCountsView()

	for _, view := range v.ChangeViews {
		resource := view.Resource()
		v.countsView.Add(view.ApplyOp(), view.WaitOp())

		row := []uitable.Value{
			uitable.NewValueString(resource.Namespace()),
			uitable.NewValueString(resource.Name()),
			uitable.NewValueString(resource.Kind()),
			uitable.NewValueString(resource.APIVersion()),
		}

		if resource.IsProvisioned() {
			condVal := cmdcore.NewConditionsValue(resource.Status())

			row = append(row,
				// TODO erroneously colors empty value
				uitable.ValueFmt{V: condVal, Error: condVal.NeedsAttention()},
				cmdcore.NewValueAge(resource.CreatedAt()),
			)
		} else {
			row = append(row,
				uitable.ValueFmt{V: uitable.NewValueString(""), Error: false},
				uitable.NewValueString(""),
			)
		}

		row = append(row,
			v.applyOpCode(view.ApplyOp()),
			v.waitOpCode(view.WaitOp()),
		)

		table.Rows = append(table.Rows, row)
	}

	table.Notes = append(table.Notes, v.countsView.Strings(true)...)

	ui.PrintTable(table)
}

func (v *ChangesView) Summary() string { return v.countsView.String() }

var (
	applyOpCodeUI = map[ClusterChangeApplyOp]string{
		ClusterChangeApplyOpAdd:    "create",
		ClusterChangeApplyOpDelete: "delete",
		ClusterChangeApplyOpUpdate: "update",
		ClusterChangeApplyOpNoop:   "noop",
	}

	waitOpCodeUI = map[ClusterChangeWaitOp]string{
		ClusterChangeWaitOpOK:     "reconcile",
		ClusterChangeWaitOpDelete: "delete",
		ClusterChangeWaitOpNoop:   "noop",
	}
)

func (v *ChangesView) applyOpCode(op ClusterChangeApplyOp) uitable.Value {
	switch op {
	case ClusterChangeApplyOpAdd:
		return uitable.ValueFmt{V: uitable.NewValueString(applyOpCodeUI[op]), Error: false}
	case ClusterChangeApplyOpDelete:
		return uitable.ValueFmt{V: uitable.NewValueString(applyOpCodeUI[op]), Error: true}
	case ClusterChangeApplyOpUpdate:
		return uitable.ValueFmt{V: uitable.NewValueString(applyOpCodeUI[op]), Error: false}
	case ClusterChangeApplyOpNoop:
		return uitable.NewValueString("")
	default:
		return uitable.NewValueString("???")
	}
}

func (v *ChangesView) waitOpCode(op ClusterChangeWaitOp) uitable.Value {
	switch op {
	case ClusterChangeWaitOpOK:
		return uitable.NewValueString(waitOpCodeUI[op]) // TODO highlight for apply op noop?
	case ClusterChangeWaitOpDelete:
		return uitable.NewValueString(waitOpCodeUI[op])
	case ClusterChangeWaitOpNoop:
		return uitable.NewValueString("")
	default:
		return uitable.NewValueString("???")
	}
}

type ChangesCountsView struct {
	applyOps map[ClusterChangeApplyOp]int
	waitOps  map[ClusterChangeWaitOp]int
}

func NewChangesCountsView() *ChangesCountsView {
	return &ChangesCountsView{map[ClusterChangeApplyOp]int{}, map[ClusterChangeWaitOp]int{}}
}

func (v *ChangesCountsView) Add(applyOp ClusterChangeApplyOp, waitOp ClusterChangeWaitOp) {
	v.applyOps[applyOp] += 1
	v.waitOps[waitOp] += 1
}

func (v *ChangesCountsView) Strings(table bool) []string {
	applyOpsStats := []string{}
	visibleApplyOps := []ClusterChangeApplyOp{
		ClusterChangeApplyOpAdd, ClusterChangeApplyOpDelete, ClusterChangeApplyOpUpdate, ClusterChangeApplyOpNoop}

	for _, op := range visibleApplyOps {
		applyOpsStats = append(applyOpsStats, fmt.Sprintf("%d %s", v.applyOps[op], applyOpCodeUI[op]))
	}

	waitsOpStats := []string{}
	visibleWaitOps := []ClusterChangeWaitOp{
		ClusterChangeWaitOpOK, ClusterChangeWaitOpDelete, ClusterChangeWaitOpNoop}

	for _, op := range visibleWaitOps {
		waitsOpStats = append(waitsOpStats, fmt.Sprintf("%d %s", v.waitOps[op], waitOpCodeUI[op]))
	}

	padding := ""
	if table {
		padding = "     "
	}

	return []string{
		"Op: " + padding + strings.Join(applyOpsStats, ", "),
		"Wait to: " + strings.Join(waitsOpStats, ", "),
	}
}

func (v *ChangesCountsView) String() string {
	return strings.Join(v.Strings(false), " / ")
}
