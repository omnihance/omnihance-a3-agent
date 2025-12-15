package echarts

import (
	"github.com/omnihance/omnihance-a3-agent/internal/utils"
)

func NewTitle() *Title {
	return &Title{}
}

func (t *Title) WithText(text string) *Title {
	t.Text = text
	return t
}

func (t *Title) WithSubtext(subtext string) *Title {
	t.Subtext = subtext
	return t
}

func (t *Title) WithLeft(left interface{}) *Title {
	t.Left = left
	return t
}

func (t *Title) WithTop(top interface{}) *Title {
	t.Top = top
	return t
}

func (t *Title) WithShow(show bool) *Title {
	t.Show = utils.BoolPtr(show)
	return t
}

func NewTooltip() *Tooltip {
	return &Tooltip{}
}

func (t *Tooltip) WithTrigger(trigger string) *Tooltip {
	t.Trigger = trigger
	return t
}

func (t *Tooltip) WithAxisPointer(axisPointer *AxisPointer) *Tooltip {
	t.AxisPointer = axisPointer
	return t
}

func (t *Tooltip) WithShow(show bool) *Tooltip {
	t.Show = utils.BoolPtr(show)
	return t
}

func (t *Tooltip) WithFormatter(formatter interface{}) *Tooltip {
	t.Formatter = formatter
	return t
}

func NewAxisPointer() *AxisPointer {
	return &AxisPointer{}
}

func (ap *AxisPointer) WithType(typ string) *AxisPointer {
	ap.Type = typ
	return ap
}

func (ap *AxisPointer) WithShow(show bool) *AxisPointer {
	ap.Show = utils.BoolPtr(show)
	return ap
}

func NewLegend() *Legend {
	return &Legend{}
}

func (l *Legend) WithShow(show bool) *Legend {
	l.Show = utils.BoolPtr(show)
	return l
}

func (l *Legend) WithData(data interface{}) *Legend {
	l.Data = data
	return l
}

func (l *Legend) WithTop(top interface{}) *Legend {
	l.Top = top
	return l
}

func (l *Legend) WithLeft(left interface{}) *Legend {
	l.Left = left
	return l
}

func (l *Legend) WithRight(right interface{}) *Legend {
	l.Right = right
	return l
}

func (l *Legend) WithBottom(bottom interface{}) *Legend {
	l.Bottom = bottom
	return l
}

func (l *Legend) WithOrient(orient string) *Legend {
	l.Orient = orient
	return l
}

func NewGrid() *Grid {
	return &Grid{}
}

func (g *Grid) WithLeft(left interface{}) *Grid {
	g.Left = left
	return g
}

func (g *Grid) WithTop(top interface{}) *Grid {
	g.Top = top
	return g
}

func (g *Grid) WithRight(right interface{}) *Grid {
	g.Right = right
	return g
}

func (g *Grid) WithBottom(bottom interface{}) *Grid {
	g.Bottom = bottom
	return g
}

func (g *Grid) WithContainLabel(containLabel bool) *Grid {
	g.ContainLabel = utils.BoolPtr(containLabel)
	return g
}

func (g *Grid) WithShow(show bool) *Grid {
	g.Show = utils.BoolPtr(show)
	return g
}

func NewAxis() *Axis {
	return &Axis{}
}

func (a *Axis) WithType(typ string) *Axis {
	a.Type = typ
	return a
}

func (a *Axis) WithName(name string) *Axis {
	a.Name = name
	return a
}

func (a *Axis) WithNameLocation(location string) *Axis {
	a.NameLocation = location
	return a
}

func (a *Axis) WithMin(min interface{}) *Axis {
	a.Min = min
	return a
}

func (a *Axis) WithMax(max interface{}) *Axis {
	a.Max = max
	return a
}

func (a *Axis) WithData(data interface{}) *Axis {
	a.Data = data
	return a
}

func (a *Axis) WithShow(show bool) *Axis {
	a.Show = utils.BoolPtr(show)
	return a
}

func (a *Axis) WithPosition(position string) *Axis {
	a.Position = position
	return a
}

func (a *Axis) WithAxisLabel(axisLabel *AxisLabel) *Axis {
	a.AxisLabel = axisLabel
	return a
}

func (a *Axis) WithAxisLine(axisLine *AxisLine) *Axis {
	a.AxisLine = axisLine
	return a
}

func (a *Axis) WithSplitLine(splitLine *SplitLine) *Axis {
	a.SplitLine = splitLine
	return a
}

func (a *Axis) WithSplitArea(splitArea *SplitArea) *Axis {
	a.SplitArea = splitArea
	return a
}

func NewAxisLabel() *AxisLabel {
	return &AxisLabel{}
}

func (al *AxisLabel) WithShow(show bool) *AxisLabel {
	al.Show = utils.BoolPtr(show)
	return al
}

func (al *AxisLabel) WithFormatter(formatter interface{}) *AxisLabel {
	al.Formatter = formatter
	return al
}

func (al *AxisLabel) WithRotate(rotate interface{}) *AxisLabel {
	al.Rotate = rotate
	return al
}

func (al *AxisLabel) WithColor(color string) *AxisLabel {
	al.Color = color
	return al
}

func (al *AxisLabel) WithFontSize(fontSize int) *AxisLabel {
	al.FontSize = utils.IntPtr(fontSize)
	return al
}

func NewAxisLine() *AxisLine {
	return &AxisLine{}
}

func (al *AxisLine) WithShow(show bool) *AxisLine {
	al.Show = utils.BoolPtr(show)
	return al
}

func (al *AxisLine) WithLineStyle(lineStyle *LineStyle) *AxisLine {
	al.LineStyle = lineStyle
	return al
}

func NewSplitLine() *SplitLine {
	return &SplitLine{}
}

func (sl *SplitLine) WithShow(show bool) *SplitLine {
	sl.Show = utils.BoolPtr(show)
	return sl
}

func (sl *SplitLine) WithLineStyle(lineStyle *LineStyle) *SplitLine {
	sl.LineStyle = lineStyle
	return sl
}

func NewSplitArea() *SplitArea {
	return &SplitArea{}
}

func (sa *SplitArea) WithShow(show bool) *SplitArea {
	sa.Show = utils.BoolPtr(show)
	return sa
}

func NewSeries() *Series {
	return &Series{}
}

func (s *Series) WithType(typ string) *Series {
	s.Type = typ
	return s
}

func (s *Series) WithName(name string) *Series {
	s.Name = name
	return s
}

func (s *Series) WithData(data interface{}) *Series {
	s.Data = data
	return s
}

func (s *Series) WithSmooth(smooth bool) *Series {
	s.Smooth = utils.BoolPtr(smooth)
	return s
}

func (s *Series) WithStack(stack string) *Series {
	s.Stack = stack
	return s
}

func (s *Series) WithSymbol(symbol interface{}) *Series {
	s.Symbol = symbol
	return s
}

func (s *Series) WithSymbolSize(symbolSize interface{}) *Series {
	s.SymbolSize = symbolSize
	return s
}

func (s *Series) WithShowSymbol(showSymbol bool) *Series {
	s.ShowSymbol = utils.BoolPtr(showSymbol)
	return s
}

func (s *Series) WithLineStyle(lineStyle *LineStyle) *Series {
	s.LineStyle = lineStyle
	return s
}

func (s *Series) WithAreaStyle(areaStyle *AreaStyle) *Series {
	s.AreaStyle = areaStyle
	return s
}

func (s *Series) WithItemStyle(itemStyle *ItemStyle) *Series {
	s.ItemStyle = itemStyle
	return s
}

func (s *Series) WithLabel(label *Label) *Series {
	s.Label = label
	return s
}

func (s *Series) WithLabelLine(labelLine *LabelLine) *Series {
	s.LabelLine = labelLine
	return s
}

func (s *Series) WithEmphasis(emphasis *Emphasis) *Series {
	s.Emphasis = emphasis
	return s
}

func (s *Series) WithXAxisIndex(index int) *Series {
	s.XAxisIndex = utils.IntPtr(index)
	return s
}

func (s *Series) WithYAxisIndex(index int) *Series {
	s.YAxisIndex = utils.IntPtr(index)
	return s
}

func (s *Series) WithRadius(radius interface{}) *Series {
	s.Radius = radius
	return s
}

func (s *Series) WithCenter(center interface{}) *Series {
	s.Center = center
	return s
}

func (s *Series) WithRoseType(roseType interface{}) *Series {
	s.RoseType = roseType
	return s
}

func (s *Series) WithBarWidth(barWidth interface{}) *Series {
	s.BarWidth = barWidth
	return s
}

func (s *Series) WithBarGap(barGap string) *Series {
	s.BarGap = barGap
	return s
}

func (s *Series) WithBarCategoryGap(barCategoryGap string) *Series {
	s.BarCategoryGap = barCategoryGap
	return s
}

func NewLineStyle() *LineStyle {
	return &LineStyle{}
}

func (ls *LineStyle) WithColor(color interface{}) *LineStyle {
	ls.Color = color
	return ls
}

func (ls *LineStyle) WithWidth(width int) *LineStyle {
	ls.Width = utils.IntPtr(width)
	return ls
}

func (ls *LineStyle) WithType(typ string) *LineStyle {
	ls.Type = typ
	return ls
}

func (ls *LineStyle) WithOpacity(opacity float64) *LineStyle {
	ls.Opacity = utils.Float64Ptr(opacity)
	return ls
}

func NewAreaStyle() *AreaStyle {
	return &AreaStyle{}
}

func (as *AreaStyle) WithColor(color interface{}) *AreaStyle {
	as.Color = color
	return as
}

func (as *AreaStyle) WithOpacity(opacity float64) *AreaStyle {
	as.Opacity = utils.Float64Ptr(opacity)
	return as
}

func NewItemStyle() *ItemStyle {
	return &ItemStyle{}
}

func (is *ItemStyle) WithColor(color interface{}) *ItemStyle {
	is.Color = color
	return is
}

func (is *ItemStyle) WithBorderColor(borderColor string) *ItemStyle {
	is.BorderColor = borderColor
	return is
}

func (is *ItemStyle) WithBorderWidth(borderWidth int) *ItemStyle {
	is.BorderWidth = utils.IntPtr(borderWidth)
	return is
}

func (is *ItemStyle) WithBorderRadius(borderRadius interface{}) *ItemStyle {
	is.BorderRadius = borderRadius
	return is
}

func (is *ItemStyle) WithOpacity(opacity float64) *ItemStyle {
	is.Opacity = utils.Float64Ptr(opacity)
	return is
}

func NewLabel() *Label {
	return &Label{}
}

func (l *Label) WithShow(show bool) *Label {
	l.Show = utils.BoolPtr(show)
	return l
}

func (l *Label) WithPosition(position interface{}) *Label {
	l.Position = position
	return l
}

func (l *Label) WithFormatter(formatter interface{}) *Label {
	l.Formatter = formatter
	return l
}

func (l *Label) WithColor(color string) *Label {
	l.Color = color
	return l
}

func (l *Label) WithFontSize(fontSize int) *Label {
	l.FontSize = utils.IntPtr(fontSize)
	return l
}

func NewLabelLine() *LabelLine {
	return &LabelLine{}
}

func (ll *LabelLine) WithShow(show bool) *LabelLine {
	ll.Show = utils.BoolPtr(show)
	return ll
}

func (ll *LabelLine) WithSmooth(smooth bool) *LabelLine {
	ll.Smooth = utils.BoolPtr(smooth)
	return ll
}

func (ll *LabelLine) WithLength(length interface{}) *LabelLine {
	ll.Length = length
	return ll
}

func NewEmphasis() *Emphasis {
	return &Emphasis{}
}

func (e *Emphasis) WithLabel(label *Label) *Emphasis {
	e.Label = label
	return e
}

func (e *Emphasis) WithItemStyle(itemStyle *ItemStyle) *Emphasis {
	e.ItemStyle = itemStyle
	return e
}

func (e *Emphasis) WithFocus(focus string) *Emphasis {
	e.Focus = focus
	return e
}

func NewTextStyle() *TextStyle {
	return &TextStyle{}
}

func (ts *TextStyle) WithColor(color string) *TextStyle {
	ts.Color = color
	return ts
}

func (ts *TextStyle) WithFontSize(fontSize int) *TextStyle {
	ts.FontSize = utils.IntPtr(fontSize)
	return ts
}

func (ts *TextStyle) WithFontWeight(fontWeight interface{}) *TextStyle {
	ts.FontWeight = fontWeight
	return ts
}

func (ts *TextStyle) WithFontFamily(fontFamily string) *TextStyle {
	ts.FontFamily = fontFamily
	return ts
}
