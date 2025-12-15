package echarts

type Option struct {
	Title                   *Title          `json:"title,omitempty"`
	Tooltip                 *Tooltip        `json:"tooltip,omitempty"`
	Legend                  *Legend         `json:"legend,omitempty"`
	Grid                    *Grid           `json:"grid,omitempty"`
	XAxis                   []*Axis         `json:"xAxis,omitempty"`
	YAxis                   []*Axis         `json:"yAxis,omitempty"`
	Polar                   interface{}     `json:"polar,omitempty"`
	RadiusAxis              interface{}     `json:"radiusAxis,omitempty"`
	AngleAxis               interface{}     `json:"angleAxis,omitempty"`
	Radar                   interface{}     `json:"radar,omitempty"`
	DataZoom                interface{}     `json:"dataZoom,omitempty"`
	VisualMap               interface{}     `json:"visualMap,omitempty"`
	Timeline                interface{}     `json:"timeline,omitempty"`
	Graphic                 interface{}     `json:"graphic,omitempty"`
	Calendar                interface{}     `json:"calendar,omitempty"`
	Dataset                 *Dataset        `json:"dataset,omitempty"`
	Series                  []*Series       `json:"series,omitempty"`
	Color                   []string        `json:"color,omitempty"`
	BackgroundColor         string          `json:"backgroundColor,omitempty"`
	TextStyle               *TextStyle      `json:"textStyle,omitempty"`
	Animation               *bool           `json:"animation,omitempty"`
	AnimationThreshold      *int            `json:"animationThreshold,omitempty"`
	AnimationDuration       *int            `json:"animationDuration,omitempty"`
	AnimationEasing         string          `json:"animationEasing,omitempty"`
	AnimationDelay          *int            `json:"animationDelay,omitempty"`
	AnimationDurationUpdate *int            `json:"animationDurationUpdate,omitempty"`
	AnimationEasingUpdate   string          `json:"animationEasingUpdate,omitempty"`
	AnimationDelayUpdate    *int            `json:"animationDelayUpdate,omitempty"`
	StateAnimation          *StateAnimation `json:"stateAnimation,omitempty"`
	UseUTC                  *bool           `json:"useUTC,omitempty"`
}

type Title struct {
	Show            *bool       `json:"show,omitempty"`
	Text            string      `json:"text,omitempty"`
	Link            string      `json:"link,omitempty"`
	Target          string      `json:"target,omitempty"`
	Subtext         string      `json:"subtext,omitempty"`
	Sublink         string      `json:"sublink,omitempty"`
	Subtarget       string      `json:"subtarget,omitempty"`
	TextStyle       *TextStyle  `json:"textStyle,omitempty"`
	SubtextStyle    *TextStyle  `json:"subtextStyle,omitempty"`
	Padding         interface{} `json:"padding,omitempty"`
	ItemGap         *int        `json:"itemGap,omitempty"`
	ZLevel          *int        `json:"zlevel,omitempty"`
	Z               *int        `json:"z,omitempty"`
	Left            interface{} `json:"left,omitempty"`
	Top             interface{} `json:"top,omitempty"`
	Right           interface{} `json:"right,omitempty"`
	Bottom          interface{} `json:"bottom,omitempty"`
	BackgroundColor string      `json:"backgroundColor,omitempty"`
	BorderColor     string      `json:"borderColor,omitempty"`
	BorderWidth     *int        `json:"borderWidth,omitempty"`
	BorderRadius    interface{} `json:"borderRadius,omitempty"`
	ShadowBlur      *int        `json:"shadowBlur,omitempty"`
	ShadowColor     string      `json:"shadowColor,omitempty"`
	ShadowOffsetX   *int        `json:"shadowOffsetX,omitempty"`
	ShadowOffsetY   *int        `json:"shadowOffsetY,omitempty"`
}

type Tooltip struct {
	Show               *bool        `json:"show,omitempty"`
	Trigger            string       `json:"trigger,omitempty"`
	AxisPointer        *AxisPointer `json:"axisPointer,omitempty"`
	ShowContent        *bool        `json:"showContent,omitempty"`
	AlwaysShowContent  *bool        `json:"alwaysShowContent,omitempty"`
	TriggerOn          string       `json:"triggerOn,omitempty"`
	ShowDelay          *int         `json:"showDelay,omitempty"`
	HideDelay          *int         `json:"hideDelay,omitempty"`
	Enterable          *bool        `json:"enterable,omitempty"`
	RenderMode         string       `json:"renderMode,omitempty"`
	Confine            *bool        `json:"confine,omitempty"`
	AppendToBody       *bool        `json:"appendToBody,omitempty"`
	ClassName          string       `json:"className,omitempty"`
	TransitionDuration *int         `json:"transitionDuration,omitempty"`
	Position           interface{}  `json:"position,omitempty"`
	Formatter          interface{}  `json:"formatter,omitempty"`
	FormatterParams    interface{}  `json:"formatterParams,omitempty"`
	ValueFormatter     interface{}  `json:"valueFormatter,omitempty"`
	BackgroundColor    string       `json:"backgroundColor,omitempty"`
	BorderColor        string       `json:"borderColor,omitempty"`
	BorderWidth        *int         `json:"borderWidth,omitempty"`
	Padding            interface{}  `json:"padding,omitempty"`
	TextStyle          *TextStyle   `json:"textStyle,omitempty"`
	ExtraCssText       string       `json:"extraCssText,omitempty"`
	Order              string       `json:"order,omitempty"`
}

type AxisPointer struct {
	Show                    *bool        `json:"show,omitempty"`
	Type                    string       `json:"type,omitempty"`
	Snap                    *bool        `json:"snap,omitempty"`
	Z                       *int         `json:"z,omitempty"`
	Label                   *Label       `json:"label,omitempty"`
	LineStyle               *LineStyle   `json:"lineStyle,omitempty"`
	ShadowStyle             *ShadowStyle `json:"shadowStyle,omitempty"`
	Handle                  *Handle      `json:"handle,omitempty"`
	Status                  string       `json:"status,omitempty"`
	Animation               *bool        `json:"animation,omitempty"`
	AnimationDurationUpdate *int         `json:"animationDurationUpdate,omitempty"`
	AnimationEasingUpdate   string       `json:"animationEasingUpdate,omitempty"`
	Link                    interface{}  `json:"link,omitempty"`
	TriggerOn               string       `json:"triggerOn,omitempty"`
	TriggerTooltip          *bool        `json:"triggerTooltip,omitempty"`
	Value                   interface{}  `json:"value,omitempty"`
}

type Legend struct {
	Show                    *bool           `json:"show,omitempty"`
	ZLevel                  *int            `json:"zlevel,omitempty"`
	Z                       *int            `json:"z,omitempty"`
	Left                    interface{}     `json:"left,omitempty"`
	Top                     interface{}     `json:"top,omitempty"`
	Right                   interface{}     `json:"right,omitempty"`
	Bottom                  interface{}     `json:"bottom,omitempty"`
	Width                   interface{}     `json:"width,omitempty"`
	Height                  interface{}     `json:"height,omitempty"`
	Orient                  string          `json:"orient,omitempty"`
	Align                   string          `json:"align,omitempty"`
	Padding                 interface{}     `json:"padding,omitempty"`
	ItemGap                 *int            `json:"itemGap,omitempty"`
	ItemWidth               *int            `json:"itemWidth,omitempty"`
	ItemHeight              *int            `json:"itemHeight,omitempty"`
	SymbolKeepAspect        *bool           `json:"symbolKeepAspect,omitempty"`
	Formatter               interface{}     `json:"formatter,omitempty"`
	SelectedMode            interface{}     `json:"selectedMode,omitempty"`
	InactiveColor           string          `json:"inactiveColor,omitempty"`
	Selected                map[string]bool `json:"selected,omitempty"`
	TextStyle               *TextStyle      `json:"textStyle,omitempty"`
	Tooltip                 *Tooltip        `json:"tooltip,omitempty"`
	Icon                    string          `json:"icon,omitempty"`
	Data                    interface{}     `json:"data,omitempty"`
	BackgroundColor         string          `json:"backgroundColor,omitempty"`
	BorderColor             string          `json:"borderColor,omitempty"`
	BorderWidth             *int            `json:"borderWidth,omitempty"`
	BorderRadius            interface{}     `json:"borderRadius,omitempty"`
	ShadowBlur              *int            `json:"shadowBlur,omitempty"`
	ShadowColor             string          `json:"shadowColor,omitempty"`
	ShadowOffsetX           *int            `json:"shadowOffsetX,omitempty"`
	ShadowOffsetY           *int            `json:"shadowOffsetY,omitempty"`
	ScrollDataIndex         *int            `json:"scrollDataIndex,omitempty"`
	PageButtonItemGap       *int            `json:"pageButtonItemGap,omitempty"`
	PageButtonPosition      string          `json:"pageButtonPosition,omitempty"`
	PageFormatter           interface{}     `json:"pageFormatter,omitempty"`
	PageIconColor           string          `json:"pageIconColor,omitempty"`
	PageIconInactiveColor   string          `json:"pageIconInactiveColor,omitempty"`
	PageIconSize            *int            `json:"pageIconSize,omitempty"`
	PageTextStyle           *TextStyle      `json:"pageTextStyle,omitempty"`
	Animation               *bool           `json:"animation,omitempty"`
	AnimationDurationUpdate *int            `json:"animationDurationUpdate,omitempty"`
}

type Grid struct {
	Show            *bool       `json:"show,omitempty"`
	ZLevel          *int        `json:"zlevel,omitempty"`
	Z               *int        `json:"z,omitempty"`
	Left            interface{} `json:"left,omitempty"`
	Top             interface{} `json:"top,omitempty"`
	Right           interface{} `json:"right,omitempty"`
	Bottom          interface{} `json:"bottom,omitempty"`
	Width           interface{} `json:"width,omitempty"`
	Height          interface{} `json:"height,omitempty"`
	ContainLabel    *bool       `json:"containLabel,omitempty"`
	BackgroundColor string      `json:"backgroundColor,omitempty"`
	BorderColor     string      `json:"borderColor,omitempty"`
	BorderWidth     *int        `json:"borderWidth,omitempty"`
	ShadowBlur      *int        `json:"shadowBlur,omitempty"`
	ShadowColor     string      `json:"shadowColor,omitempty"`
	ShadowOffsetX   *int        `json:"shadowOffsetX,omitempty"`
	ShadowOffsetY   *int        `json:"shadowOffsetY,omitempty"`
	Tooltip         *Tooltip    `json:"tooltip,omitempty"`
}

type Axis struct {
	Show          *bool        `json:"show,omitempty"`
	GridIndex     *int         `json:"gridIndex,omitempty"`
	Position      string       `json:"position,omitempty"`
	Offset        *int         `json:"offset,omitempty"`
	Type          string       `json:"type,omitempty"`
	Name          string       `json:"name,omitempty"`
	NameLocation  string       `json:"nameLocation,omitempty"`
	NameTextStyle *TextStyle   `json:"nameTextStyle,omitempty"`
	NameGap       *int         `json:"nameGap,omitempty"`
	NameRotate    interface{}  `json:"nameRotate,omitempty"`
	Inverse       *bool        `json:"inverse,omitempty"`
	BoundaryGap   interface{}  `json:"boundaryGap,omitempty"`
	Min           interface{}  `json:"min,omitempty"`
	Max           interface{}  `json:"max,omitempty"`
	Scale         *bool        `json:"scale,omitempty"`
	SplitNumber   *int         `json:"splitNumber,omitempty"`
	MinInterval   interface{}  `json:"minInterval,omitempty"`
	MaxInterval   interface{}  `json:"maxInterval,omitempty"`
	Interval      interface{}  `json:"interval,omitempty"`
	LogBase       *int         `json:"logBase,omitempty"`
	Silent        *bool        `json:"silent,omitempty"`
	TriggerEvent  *bool        `json:"triggerEvent,omitempty"`
	AxisLine      *AxisLine    `json:"axisLine,omitempty"`
	AxisTick      *AxisTick    `json:"axisTick,omitempty"`
	MinorTick     *MinorTick   `json:"minorTick,omitempty"`
	SplitLine     *SplitLine   `json:"splitLine,omitempty"`
	SplitArea     *SplitArea   `json:"splitArea,omitempty"`
	Data          interface{}  `json:"data,omitempty"`
	AxisLabel     *AxisLabel   `json:"axisLabel,omitempty"`
	AxisPointer   *AxisPointer `json:"axisPointer,omitempty"`
	ZLevel        *int         `json:"zlevel,omitempty"`
	Z             *int         `json:"z,omitempty"`
}

type AxisLine struct {
	Show            *bool       `json:"show,omitempty"`
	OnZero          *bool       `json:"onZero,omitempty"`
	OnZeroAxisIndex interface{} `json:"onZeroAxisIndex,omitempty"`
	Symbol          interface{} `json:"symbol,omitempty"`
	SymbolSize      interface{} `json:"symbolSize,omitempty"`
	SymbolOffset    interface{} `json:"symbolOffset,omitempty"`
	LineStyle       *LineStyle  `json:"lineStyle,omitempty"`
}

type AxisTick struct {
	Show           *bool      `json:"show,omitempty"`
	AlignWithLabel *bool      `json:"alignWithLabel,omitempty"`
	Inside         *bool      `json:"inside,omitempty"`
	Length         *int       `json:"length,omitempty"`
	LineStyle      *LineStyle `json:"lineStyle,omitempty"`
}

type MinorTick struct {
	Show        *bool      `json:"show,omitempty"`
	SplitNumber *int       `json:"splitNumber,omitempty"`
	Length      *int       `json:"length,omitempty"`
	LineStyle   *LineStyle `json:"lineStyle,omitempty"`
}

type SplitLine struct {
	Show      *bool       `json:"show,omitempty"`
	Interval  interface{} `json:"interval,omitempty"`
	LineStyle *LineStyle  `json:"lineStyle,omitempty"`
}

type SplitArea struct {
	Show      *bool       `json:"show,omitempty"`
	Interval  interface{} `json:"interval,omitempty"`
	AreaStyle *AreaStyle  `json:"areaStyle,omitempty"`
}

type AxisLabel struct {
	Show                 *bool                  `json:"show,omitempty"`
	Interval             interface{}            `json:"interval,omitempty"`
	Inside               *bool                  `json:"inside,omitempty"`
	Rotate               interface{}            `json:"rotate,omitempty"`
	Margin               *int                   `json:"margin,omitempty"`
	Formatter            interface{}            `json:"formatter,omitempty"`
	ShowMinLabel         *bool                  `json:"showMinLabel,omitempty"`
	ShowMaxLabel         *bool                  `json:"showMaxLabel,omitempty"`
	HideOverlap          *bool                  `json:"hideOverlap,omitempty"`
	Color                string                 `json:"color,omitempty"`
	FontStyle            string                 `json:"fontStyle,omitempty"`
	FontWeight           interface{}            `json:"fontWeight,omitempty"`
	FontFamily           string                 `json:"fontFamily,omitempty"`
	FontSize             *int                   `json:"fontSize,omitempty"`
	Align                string                 `json:"align,omitempty"`
	VerticalAlign        string                 `json:"verticalAlign,omitempty"`
	LineHeight           interface{}            `json:"lineHeight,omitempty"`
	BackgroundColor      string                 `json:"backgroundColor,omitempty"`
	BorderColor          string                 `json:"borderColor,omitempty"`
	BorderWidth          *int                   `json:"borderWidth,omitempty"`
	BorderRadius         interface{}            `json:"borderRadius,omitempty"`
	Padding              interface{}            `json:"padding,omitempty"`
	ShadowBlur           *int                   `json:"shadowBlur,omitempty"`
	ShadowColor          string                 `json:"shadowColor,omitempty"`
	ShadowOffsetX        *int                   `json:"shadowOffsetX,omitempty"`
	ShadowOffsetY        *int                   `json:"shadowOffsetY,omitempty"`
	Width                interface{}            `json:"width,omitempty"`
	Height               interface{}            `json:"height,omitempty"`
	TextBorderColor      string                 `json:"textBorderColor,omitempty"`
	TextBorderWidth      *int                   `json:"textBorderWidth,omitempty"`
	TextBorderType       string                 `json:"textBorderType,omitempty"`
	TextBorderDashOffset *int                   `json:"textBorderDashOffset,omitempty"`
	TextShadowBlur       *int                   `json:"textShadowBlur,omitempty"`
	TextShadowColor      string                 `json:"textShadowColor,omitempty"`
	TextShadowOffsetX    *int                   `json:"textShadowOffsetX,omitempty"`
	TextShadowOffsetY    *int                   `json:"textShadowOffsetY,omitempty"`
	Rich                 map[string]interface{} `json:"rich,omitempty"`
}

type Series struct {
	Type                    string      `json:"type,omitempty"`
	Id                      string      `json:"id,omitempty"`
	Name                    string      `json:"name,omitempty"`
	ColorBy                 string      `json:"colorBy,omitempty"`
	LegendHoverLink         *bool       `json:"legendHoverLink,omitempty"`
	CoordinateSystem        string      `json:"coordinateSystem,omitempty"`
	XAxisIndex              *int        `json:"xAxisIndex,omitempty"`
	YAxisIndex              *int        `json:"yAxisIndex,omitempty"`
	PolarIndex              *int        `json:"polarIndex,omitempty"`
	GeoIndex                *int        `json:"geoIndex,omitempty"`
	CalendarIndex           *int        `json:"calendarIndex,omitempty"`
	Data                    interface{} `json:"data,omitempty"`
	DatasetIndex            *int        `json:"datasetIndex,omitempty"`
	SourceHeader            *bool       `json:"sourceHeader,omitempty"`
	Encode                  *Encode     `json:"encode,omitempty"`
	SeriesLayoutBy          string      `json:"seriesLayoutBy,omitempty"`
	ZLevel                  *int        `json:"zlevel,omitempty"`
	Z                       *int        `json:"z,omitempty"`
	Silent                  *bool       `json:"silent,omitempty"`
	Animation               *bool       `json:"animation,omitempty"`
	AnimationThreshold      *int        `json:"animationThreshold,omitempty"`
	AnimationDuration       interface{} `json:"animationDuration,omitempty"`
	AnimationEasing         string      `json:"animationEasing,omitempty"`
	AnimationDelay          interface{} `json:"animationDelay,omitempty"`
	AnimationDurationUpdate interface{} `json:"animationDurationUpdate,omitempty"`
	AnimationEasingUpdate   string      `json:"animationEasingUpdate,omitempty"`
	AnimationDelayUpdate    interface{} `json:"animationDelayUpdate,omitempty"`
	Tooltip                 *Tooltip    `json:"tooltip,omitempty"`

	LineStyle *LineStyle `json:"lineStyle,omitempty"`
	AreaStyle *AreaStyle `json:"areaStyle,omitempty"`
	ItemStyle *ItemStyle `json:"itemStyle,omitempty"`
	Label     *Label     `json:"label,omitempty"`
	LabelLine *LabelLine `json:"labelLine,omitempty"`
	Emphasis  *Emphasis  `json:"emphasis,omitempty"`
	Blur      *Blur      `json:"blur,omitempty"`
	Select    *Select    `json:"select,omitempty"`
	MarkPoint *MarkPoint `json:"markPoint,omitempty"`
	MarkLine  *MarkLine  `json:"markLine,omitempty"`
	MarkArea  *MarkArea  `json:"markArea,omitempty"`

	Smooth           *bool       `json:"smooth,omitempty"`
	Step             interface{} `json:"step,omitempty"`
	Stack            string      `json:"stack,omitempty"`
	Symbol           interface{} `json:"symbol,omitempty"`
	SymbolSize       interface{} `json:"symbolSize,omitempty"`
	SymbolRotate     interface{} `json:"symbolRotate,omitempty"`
	SymbolOffset     interface{} `json:"symbolOffset,omitempty"`
	SymbolKeepAspect *bool       `json:"symbolKeepAspect,omitempty"`
	ShowSymbol       *bool       `json:"showSymbol,omitempty"`
	ShowAllSymbol    *bool       `json:"showAllSymbol,omitempty"`
	HoverAnimation   *bool       `json:"hoverAnimation,omitempty"`
	ConnectNulls     *bool       `json:"connectNulls,omitempty"`
	Clip             *bool       `json:"clip,omitempty"`

	BarWidth       interface{} `json:"barWidth,omitempty"`
	BarMaxWidth    interface{} `json:"barMaxWidth,omitempty"`
	BarMinWidth    interface{} `json:"barMinWidth,omitempty"`
	BarGap         string      `json:"barGap,omitempty"`
	BarCategoryGap string      `json:"barCategoryGap,omitempty"`

	Radius            interface{} `json:"radius,omitempty"`
	Center            interface{} `json:"center,omitempty"`
	RoseType          interface{} `json:"roseType,omitempty"`
	AvoidLabelOverlap *bool       `json:"avoidLabelOverlap,omitempty"`
	StillShowZeroSum  *bool       `json:"stillShowZeroSum,omitempty"`
	PercentPrecision  *int        `json:"percentPrecision,omitempty"`
	MinAngle          *int        `json:"minAngle,omitempty"`
	MinShowLabelAngle *int        `json:"minShowLabelAngle,omitempty"`
	Clockwise         *bool       `json:"clockwise,omitempty"`
	StartAngle        *int        `json:"startAngle,omitempty"`
	EndAngle          interface{} `json:"endAngle,omitempty"`

	Large                *bool  `json:"large,omitempty"`
	LargeThreshold       *int   `json:"largeThreshold,omitempty"`
	Progressive          *int   `json:"progressive,omitempty"`
	ProgressiveThreshold *int   `json:"progressiveThreshold,omitempty"`
	ProgressiveChunkMode string `json:"progressiveChunkMode,omitempty"`

	Sampling   string      `json:"sampling,omitempty"`
	Dimensions interface{} `json:"dimensions,omitempty"`
}

type Dataset struct {
	Source       interface{} `json:"source,omitempty"`
	SourceHeader *bool       `json:"sourceHeader,omitempty"`
	Dimensions   interface{} `json:"dimensions,omitempty"`
}

type Encode struct {
	X          interface{} `json:"x,omitempty"`
	Y          interface{} `json:"y,omitempty"`
	Value      interface{} `json:"value,omitempty"`
	ItemName   interface{} `json:"itemName,omitempty"`
	SeriesName interface{} `json:"seriesName,omitempty"`
}

type LineStyle struct {
	Color         interface{} `json:"color,omitempty"`
	Width         *int        `json:"width,omitempty"`
	Type          string      `json:"type,omitempty"`
	DashOffset    *int        `json:"dashOffset,omitempty"`
	Cap           string      `json:"cap,omitempty"`
	Join          string      `json:"join,omitempty"`
	MiterLimit    *int        `json:"miterLimit,omitempty"`
	ShadowBlur    *int        `json:"shadowBlur,omitempty"`
	ShadowColor   string      `json:"shadowColor,omitempty"`
	ShadowOffsetX *int        `json:"shadowOffsetX,omitempty"`
	ShadowOffsetY *int        `json:"shadowOffsetY,omitempty"`
	Opacity       *float64    `json:"opacity,omitempty"`
	Curveness     *float64    `json:"curveness,omitempty"`
}

type AreaStyle struct {
	Color         interface{} `json:"color,omitempty"`
	ShadowBlur    *int        `json:"shadowBlur,omitempty"`
	ShadowColor   string      `json:"shadowColor,omitempty"`
	ShadowOffsetX *int        `json:"shadowOffsetX,omitempty"`
	ShadowOffsetY *int        `json:"shadowOffsetY,omitempty"`
	Opacity       *float64    `json:"opacity,omitempty"`
}

type ItemStyle struct {
	Color            interface{} `json:"color,omitempty"`
	BorderColor      string      `json:"borderColor,omitempty"`
	BorderWidth      *int        `json:"borderWidth,omitempty"`
	BorderType       string      `json:"borderType,omitempty"`
	BorderDashOffset *int        `json:"borderDashOffset,omitempty"`
	BorderRadius     interface{} `json:"borderRadius,omitempty"`
	ShadowBlur       *int        `json:"shadowBlur,omitempty"`
	ShadowColor      string      `json:"shadowColor,omitempty"`
	ShadowOffsetX    *int        `json:"shadowOffsetX,omitempty"`
	ShadowOffsetY    *int        `json:"shadowOffsetY,omitempty"`
	Opacity          *float64    `json:"opacity,omitempty"`
}

type Label struct {
	Show                 *bool                  `json:"show,omitempty"`
	Position             interface{}            `json:"position,omitempty"`
	Distance             interface{}            `json:"distance,omitempty"`
	Rotate               interface{}            `json:"rotate,omitempty"`
	Offset               interface{}            `json:"offset,omitempty"`
	MinMargin            *int                   `json:"minMargin,omitempty"`
	Overflow             string                 `json:"overflow,omitempty"`
	Silent               *bool                  `json:"silent,omitempty"`
	Precision            interface{}            `json:"precision,omitempty"`
	ValueAnimation       *bool                  `json:"valueAnimation,omitempty"`
	Rich                 map[string]interface{} `json:"rich,omitempty"`
	Formatter            interface{}            `json:"formatter,omitempty"`
	Color                string                 `json:"color,omitempty"`
	FontStyle            string                 `json:"fontStyle,omitempty"`
	FontWeight           interface{}            `json:"fontWeight,omitempty"`
	FontFamily           string                 `json:"fontFamily,omitempty"`
	FontSize             *int                   `json:"fontSize,omitempty"`
	Align                string                 `json:"align,omitempty"`
	VerticalAlign        string                 `json:"verticalAlign,omitempty"`
	LineHeight           interface{}            `json:"lineHeight,omitempty"`
	BackgroundColor      string                 `json:"backgroundColor,omitempty"`
	BorderColor          string                 `json:"borderColor,omitempty"`
	BorderWidth          *int                   `json:"borderWidth,omitempty"`
	BorderRadius         interface{}            `json:"borderRadius,omitempty"`
	Padding              interface{}            `json:"padding,omitempty"`
	ShadowBlur           *int                   `json:"shadowBlur,omitempty"`
	ShadowColor          string                 `json:"shadowColor,omitempty"`
	ShadowOffsetX        *int                   `json:"shadowOffsetX,omitempty"`
	ShadowOffsetY        *int                   `json:"shadowOffsetY,omitempty"`
	Width                interface{}            `json:"width,omitempty"`
	Height               interface{}            `json:"height,omitempty"`
	TextBorderColor      string                 `json:"textBorderColor,omitempty"`
	TextBorderWidth      *int                   `json:"textBorderWidth,omitempty"`
	TextBorderType       string                 `json:"textBorderType,omitempty"`
	TextBorderDashOffset *int                   `json:"textBorderDashOffset,omitempty"`
	TextShadowBlur       *int                   `json:"textShadowBlur,omitempty"`
	TextShadowColor      string                 `json:"textShadowColor,omitempty"`
	TextShadowOffsetX    *int                   `json:"textShadowOffsetX,omitempty"`
	TextShadowOffsetY    *int                   `json:"textShadowOffsetY,omitempty"`
}

type LabelLine struct {
	Show         *bool       `json:"show,omitempty"`
	ShowAbove    *bool       `json:"showAbove,omitempty"`
	Length       interface{} `json:"length,omitempty"`
	Length2      interface{} `json:"length2,omitempty"`
	Smooth       *bool       `json:"smooth,omitempty"`
	MinTurnAngle *int        `json:"minTurnAngle,omitempty"`
	LineStyle    *LineStyle  `json:"lineStyle,omitempty"`
	Label        *Label      `json:"label,omitempty"`
}

type Emphasis struct {
	Disabled  *bool      `json:"disabled,omitempty"`
	Scale     *bool      `json:"scale,omitempty"`
	Focus     string     `json:"focus,omitempty"`
	BlurScope string     `json:"blurScope,omitempty"`
	Label     *Label     `json:"label,omitempty"`
	LabelLine *LabelLine `json:"labelLine,omitempty"`
	ItemStyle *ItemStyle `json:"itemStyle,omitempty"`
	LineStyle *LineStyle `json:"lineStyle,omitempty"`
	AreaStyle *AreaStyle `json:"areaStyle,omitempty"`
}

type Blur struct {
	Label     *Label     `json:"label,omitempty"`
	LabelLine *LabelLine `json:"labelLine,omitempty"`
	ItemStyle *ItemStyle `json:"itemStyle,omitempty"`
	LineStyle *LineStyle `json:"lineStyle,omitempty"`
	AreaStyle *AreaStyle `json:"areaStyle,omitempty"`
}

type Select struct {
	Disabled  *bool      `json:"disabled,omitempty"`
	Label     *Label     `json:"label,omitempty"`
	LabelLine *LabelLine `json:"labelLine,omitempty"`
	ItemStyle *ItemStyle `json:"itemStyle,omitempty"`
	LineStyle *LineStyle `json:"lineStyle,omitempty"`
	AreaStyle *AreaStyle `json:"areaStyle,omitempty"`
}

type MarkPoint struct {
	Symbol                  interface{} `json:"symbol,omitempty"`
	SymbolSize              interface{} `json:"symbolSize,omitempty"`
	SymbolRotate            interface{} `json:"symbolRotate,omitempty"`
	SymbolOffset            interface{} `json:"symbolOffset,omitempty"`
	Silent                  *bool       `json:"silent,omitempty"`
	Label                   *Label      `json:"label,omitempty"`
	ItemStyle               *ItemStyle  `json:"itemStyle,omitempty"`
	Emphasis                *Emphasis   `json:"emphasis,omitempty"`
	Blur                    *Blur       `json:"blur,omitempty"`
	Data                    interface{} `json:"data,omitempty"`
	Animation               *bool       `json:"animation,omitempty"`
	AnimationThreshold      *int        `json:"animationThreshold,omitempty"`
	AnimationDuration       interface{} `json:"animationDuration,omitempty"`
	AnimationEasing         string      `json:"animationEasing,omitempty"`
	AnimationDelay          interface{} `json:"animationDelay,omitempty"`
	AnimationDurationUpdate interface{} `json:"animationDurationUpdate,omitempty"`
	AnimationEasingUpdate   string      `json:"animationEasingUpdate,omitempty"`
	AnimationDelayUpdate    interface{} `json:"animationDelayUpdate,omitempty"`
}

type MarkLine struct {
	Silent                  *bool       `json:"silent,omitempty"`
	Symbol                  interface{} `json:"symbol,omitempty"`
	SymbolSize              interface{} `json:"symbolSize,omitempty"`
	Precision               *int        `json:"precision,omitempty"`
	Label                   *Label      `json:"label,omitempty"`
	LineStyle               *LineStyle  `json:"lineStyle,omitempty"`
	Emphasis                *Emphasis   `json:"emphasis,omitempty"`
	Blur                    *Blur       `json:"blur,omitempty"`
	Data                    interface{} `json:"data,omitempty"`
	Animation               *bool       `json:"animation,omitempty"`
	AnimationThreshold      *int        `json:"animationThreshold,omitempty"`
	AnimationDuration       interface{} `json:"animationDuration,omitempty"`
	AnimationEasing         string      `json:"animationEasing,omitempty"`
	AnimationDelay          interface{} `json:"animationDelay,omitempty"`
	AnimationDurationUpdate interface{} `json:"animationDurationUpdate,omitempty"`
	AnimationEasingUpdate   string      `json:"animationEasingUpdate,omitempty"`
	AnimationDelayUpdate    interface{} `json:"animationDelayUpdate,omitempty"`
}

type MarkArea struct {
	Silent                  *bool       `json:"silent,omitempty"`
	Label                   *Label      `json:"label,omitempty"`
	ItemStyle               *ItemStyle  `json:"itemStyle,omitempty"`
	Emphasis                *Emphasis   `json:"emphasis,omitempty"`
	Blur                    *Blur       `json:"blur,omitempty"`
	Data                    interface{} `json:"data,omitempty"`
	Animation               *bool       `json:"animation,omitempty"`
	AnimationThreshold      *int        `json:"animationThreshold,omitempty"`
	AnimationDuration       interface{} `json:"animationDuration,omitempty"`
	AnimationEasing         string      `json:"animationEasing,omitempty"`
	AnimationDelay          interface{} `json:"animationDelay,omitempty"`
	AnimationDurationUpdate interface{} `json:"animationDurationUpdate,omitempty"`
	AnimationEasingUpdate   string      `json:"animationEasingUpdate,omitempty"`
	AnimationDelayUpdate    interface{} `json:"animationDelayUpdate,omitempty"`
}

type TextStyle struct {
	Color                string                 `json:"color,omitempty"`
	FontStyle            string                 `json:"fontStyle,omitempty"`
	FontWeight           interface{}            `json:"fontWeight,omitempty"`
	FontFamily           string                 `json:"fontFamily,omitempty"`
	FontSize             *int                   `json:"fontSize,omitempty"`
	LineHeight           interface{}            `json:"lineHeight,omitempty"`
	Width                interface{}            `json:"width,omitempty"`
	Height               interface{}            `json:"height,omitempty"`
	TextBorderColor      string                 `json:"textBorderColor,omitempty"`
	TextBorderWidth      *int                   `json:"textBorderWidth,omitempty"`
	TextBorderType       string                 `json:"textBorderType,omitempty"`
	TextBorderDashOffset *int                   `json:"textBorderDashOffset,omitempty"`
	TextShadowBlur       *int                   `json:"textShadowBlur,omitempty"`
	TextShadowColor      string                 `json:"textShadowColor,omitempty"`
	TextShadowOffsetX    *int                   `json:"textShadowOffsetX,omitempty"`
	TextShadowOffsetY    *int                   `json:"textShadowOffsetY,omitempty"`
	Rich                 map[string]interface{} `json:"rich,omitempty"`
}

type ShadowStyle struct {
	Color         string   `json:"color,omitempty"`
	ShadowBlur    *int     `json:"shadowBlur,omitempty"`
	ShadowColor   string   `json:"shadowColor,omitempty"`
	ShadowOffsetX *int     `json:"shadowOffsetX,omitempty"`
	ShadowOffsetY *int     `json:"shadowOffsetY,omitempty"`
	Opacity       *float64 `json:"opacity,omitempty"`
}

type Handle struct {
	Show          *bool       `json:"show,omitempty"`
	Icon          interface{} `json:"icon,omitempty"`
	Size          *int        `json:"size,omitempty"`
	Margin        *int        `json:"margin,omitempty"`
	Color         string      `json:"color,omitempty"`
	Throttle      *int        `json:"throttle,omitempty"`
	ShadowBlur    *int        `json:"shadowBlur,omitempty"`
	ShadowColor   string      `json:"shadowColor,omitempty"`
	ShadowOffsetX *int        `json:"shadowOffsetX,omitempty"`
	ShadowOffsetY *int        `json:"shadowOffsetY,omitempty"`
}

type StateAnimation struct {
	Duration *int   `json:"duration,omitempty"`
	Easing   string `json:"easing,omitempty"`
}
