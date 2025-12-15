package echarts

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicLineChart(t *testing.T) {
	service := NewService()

	service.AddXAxis(
		NewAxis().
			WithType("category").
			WithData([]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}),
	)

	service.AddYAxis(NewAxis().WithType("value"))

	service.AddSeries(
		NewSeries().
			WithType("line").
			WithData([]interface{}{820, 932, 901, 934, 1290, 1330, 1320}),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	assert.NotNil(t, result["xAxis"])
	assert.NotNil(t, result["yAxis"])
	assert.NotNil(t, result["series"])

	xAxis := result["xAxis"].([]interface{})
	assert.Equal(t, "category", xAxis[0].(map[string]interface{})["type"])

	series := result["series"].([]interface{})
	assert.Equal(t, "line", series[0].(map[string]interface{})["type"])
}

func TestBasicBarChart(t *testing.T) {
	service := NewService()

	service.SetTooltip(NewTooltip())
	service.SetLegend(NewLegend())

	service.AddXAxis(
		NewAxis().
			WithData([]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}),
	)

	service.AddYAxis(NewAxis())

	service.AddSeries(
		NewSeries().
			WithName("Sale").
			WithType("bar").
			WithData([]interface{}{5, 20, 36, 10, 10, 20, 4}),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	assert.NotNil(t, result["tooltip"])
	assert.NotNil(t, result["legend"])

	series := result["series"].([]interface{})
	barSeries := series[0].(map[string]interface{})
	assert.Equal(t, "bar", barSeries["type"])
	assert.Equal(t, "Sale", barSeries["name"])
}

func TestMultipleSeriesBarChart(t *testing.T) {
	service := NewService()

	service.SetLegend(
		NewLegend().
			WithData([]string{"Food", "Cloth", "Book"}),
	)

	service.SetGrid(
		NewGrid().
			WithLeft("3%").
			WithRight("4%").
			WithBottom("3%").
			WithContainLabel(true),
	)

	service.AddXAxis(
		NewAxis().
			WithType("category").
			WithData([]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}),
	)

	service.AddYAxis(
		NewAxis().
			WithType("value"),
	)

	service.AddSeries(
		NewSeries().
			WithName("Food").
			WithType("bar").
			WithData([]interface{}{320, 302, 301, 334, 390, 330, 320}),
	)

	service.AddSeries(
		NewSeries().
			WithName("Cloth").
			WithType("bar").
			WithData([]interface{}{150, 212, 201, 154, 190, 330, 410}),
	)

	service.AddSeries(
		NewSeries().
			WithName("Book").
			WithType("bar").
			WithData([]interface{}{820, 832, 901, 934, 1290, 1330, 1320}),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	series := result["series"].([]interface{})
	assert.Len(t, series, 3)

	legend := result["legend"].(map[string]interface{})
	legendDataInterface := legend["data"].([]interface{})
	legendData := make([]string, len(legendDataInterface))
	for i, v := range legendDataInterface {
		legendData[i] = v.(string)
	}
	assert.Equal(t, []string{"Food", "Cloth", "Book"}, legendData)
}

func TestBasicPieChart(t *testing.T) {
	service := NewService()

	service.SetLegend(
		NewLegend().
			WithOrient("vertical").
			WithLeft("left").
			WithData([]string{"Apple", "Grapes", "Pineapples", "Oranges", "Bananas"}),
	)

	pieData := []map[string]interface{}{
		{"value": 335, "name": "Apple"},
		{"value": 310, "name": "Grapes"},
		{"value": 234, "name": "Pineapples"},
		{"value": 135, "name": "Oranges"},
		{"value": 1548, "name": "Bananas"},
	}

	service.AddSeries(
		NewSeries().
			WithType("pie").
			WithData(pieData),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	series := result["series"].([]interface{})
	pieSeries := series[0].(map[string]interface{})
	assert.Equal(t, "pie", pieSeries["type"])

	dataInterface := pieSeries["data"].([]interface{})
	data := make([]map[string]interface{}, len(dataInterface))
	for i, v := range dataInterface {
		data[i] = v.(map[string]interface{})
	}
	assert.Len(t, data, 5)
	assert.Equal(t, "Apple", data[0]["name"])
	assert.Equal(t, float64(335), data[0]["value"])
}

func TestPieChartWithRadius(t *testing.T) {
	service := NewService()

	service.AddSeries(
		NewSeries().
			WithName("Reference Page").
			WithType("pie").
			WithRadius("55%").
			WithData([]map[string]interface{}{
				{"value": 400, "name": "Searching Engine"},
				{"value": 335, "name": "Direct"},
				{"value": 310, "name": "Email"},
				{"value": 274, "name": "Alliance Advertisement"},
				{"value": 235, "name": "Video Advertisement"},
			}),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	series := result["series"].([]interface{})
	pieSeries := series[0].(map[string]interface{})
	assert.Equal(t, "55%", pieSeries["radius"])
	assert.Equal(t, "Reference Page", pieSeries["name"])
}

func TestLineChartWithSmooth(t *testing.T) {
	service := NewService()

	service.AddXAxis(
		NewAxis().
			WithType("category").
			WithData([]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}),
	)

	service.AddYAxis(NewAxis().WithType("value"))

	service.AddSeries(
		NewSeries().
			WithType("line").
			WithData([]interface{}{150, 230, 224, 218, 135, 147, 260}).
			WithSmooth(true),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	series := result["series"].([]interface{})
	lineSeries := series[0].(map[string]interface{})
	assert.True(t, lineSeries["smooth"].(bool))
}

func TestChartWithTitle(t *testing.T) {
	service := NewService()

	service.SetTitle(
		NewTitle().
			WithText("My Chart").
			WithSubtext("Subtitle").
			WithLeft("center"),
	)

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddYAxis(NewAxis().WithType("value"))
	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	result, err := service.ToMap()
	require.NoError(t, err)

	title := result["title"].(map[string]interface{})
	assert.Equal(t, "My Chart", title["text"])
	assert.Equal(t, "Subtitle", title["subtext"])
	assert.Equal(t, "center", title["left"])
}

func TestChartWithTooltip(t *testing.T) {
	service := NewService()

	service.SetTooltip(
		NewTooltip().
			WithTrigger("axis").
			WithAxisPointer(
				NewAxisPointer().
					WithType("cross"),
			),
	)

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddYAxis(NewAxis().WithType("value"))
	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	result, err := service.ToMap()
	require.NoError(t, err)

	tooltip := result["tooltip"].(map[string]interface{})
	assert.Equal(t, "axis", tooltip["trigger"])

	axisPointer := tooltip["axisPointer"].(map[string]interface{})
	assert.Equal(t, "cross", axisPointer["type"])
}

func TestChartWithGrid(t *testing.T) {
	service := NewService()

	service.SetGrid(
		NewGrid().
			WithLeft("3%").
			WithRight("4%").
			WithBottom("3%").
			WithTop("10%").
			WithContainLabel(true),
	)

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddYAxis(NewAxis().WithType("value"))
	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	result, err := service.ToMap()
	require.NoError(t, err)

	grid := result["grid"].(map[string]interface{})
	assert.Equal(t, "3%", grid["left"])
	assert.Equal(t, "4%", grid["right"])
	assert.Equal(t, "3%", grid["bottom"])
	assert.Equal(t, "10%", grid["top"])
	assert.True(t, grid["containLabel"].(bool))
}

func TestAxisWithMinMax(t *testing.T) {
	service := NewService()

	service.AddXAxis(
		NewAxis().
			WithType("value").
			WithMin(-100).
			WithMax(80),
	)

	service.AddYAxis(
		NewAxis().
			WithType("value").
			WithMin(-30).
			WithMax(60),
	)

	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	result, err := service.ToMap()
	require.NoError(t, err)

	xAxis := result["xAxis"].([]interface{})
	xAxisConfig := xAxis[0].(map[string]interface{})
	assert.Equal(t, float64(-100), xAxisConfig["min"])
	assert.Equal(t, float64(80), xAxisConfig["max"])

	yAxis := result["yAxis"].([]interface{})
	yAxisConfig := yAxis[0].(map[string]interface{})
	assert.Equal(t, float64(-30), yAxisConfig["min"])
	assert.Equal(t, float64(60), yAxisConfig["max"])
}

func TestAxisWithLabelFormatter(t *testing.T) {
	service := NewService()

	service.AddXAxis(NewAxis().WithType("category"))

	service.AddYAxis(
		NewAxis().
			WithType("value").
			WithName("Usage (%)").
			WithMin(0).
			WithMax(100).
			WithAxisLabel(
				NewAxisLabel().
					WithFormatter("{value}%"),
			),
	)

	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	result, err := service.ToMap()
	require.NoError(t, err)

	yAxis := result["yAxis"].([]interface{})
	yAxisConfig := yAxis[0].(map[string]interface{})
	assert.Equal(t, "Usage (%)", yAxisConfig["name"])

	axisLabel := yAxisConfig["axisLabel"].(map[string]interface{})
	assert.Equal(t, "{value}%", axisLabel["formatter"])
}

func TestTimeAxisChart(t *testing.T) {
	service := NewService()

	timeData := []interface{}{
		[]interface{}{1609459200000, 116},
		[]interface{}{1609545600000, 129},
		[]interface{}{1609632000000, 135},
	}

	service.AddXAxis(
		NewAxis().
			WithType("time"),
	)

	service.AddYAxis(NewAxis().WithType("value"))

	service.AddSeries(
		NewSeries().
			WithType("line").
			WithData(timeData),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	xAxis := result["xAxis"].([]interface{})
	assert.Equal(t, "time", xAxis[0].(map[string]interface{})["type"])
}

func TestChartWithBackgroundColor(t *testing.T) {
	service := NewService()

	service.SetBackgroundColor("#fff")

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddYAxis(NewAxis().WithType("value"))
	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	result, err := service.ToMap()
	require.NoError(t, err)

	assert.Equal(t, "#fff", result["backgroundColor"])
}

func TestChartWithColors(t *testing.T) {
	service := NewService()

	service.SetColors([]string{"#c23531", "#2f4554", "#61a0a8"})

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddYAxis(NewAxis().WithType("value"))
	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	result, err := service.ToMap()
	require.NoError(t, err)

	colorsInterface := result["color"].([]interface{})
	colors := make([]string, len(colorsInterface))
	for i, v := range colorsInterface {
		colors[i] = v.(string)
	}
	assert.Equal(t, []string{"#c23531", "#2f4554", "#61a0a8"}, colors)
}

func TestChartWithAnimation(t *testing.T) {
	service := NewService()

	animation := true
	service.SetAnimation(&animation)
	service.SetAnimationDuration(1000)
	service.SetAnimationEasing("cubicOut")

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddYAxis(NewAxis().WithType("value"))
	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	result, err := service.ToMap()
	require.NoError(t, err)

	assert.True(t, result["animation"].(bool))
	assert.Equal(t, float64(1000), result["animationDuration"])
	assert.Equal(t, "cubicOut", result["animationEasing"])
}

func TestSeriesWithSymbol(t *testing.T) {
	service := NewService()

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddYAxis(NewAxis().WithType("value"))

	service.AddSeries(
		NewSeries().
			WithType("line").
			WithData([]interface{}{1, 2, 3}).
			WithSymbol("circle").
			WithSymbolSize(20).
			WithShowSymbol(true),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	series := result["series"].([]interface{})
	lineSeries := series[0].(map[string]interface{})
	assert.Equal(t, "circle", lineSeries["symbol"])
	assert.Equal(t, float64(20), lineSeries["symbolSize"])
	assert.True(t, lineSeries["showSymbol"].(bool))
}

func TestSeriesWithLineStyle(t *testing.T) {
	service := NewService()

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddYAxis(NewAxis().WithType("value"))

	service.AddSeries(
		NewSeries().
			WithType("line").
			WithData([]interface{}{1, 2, 3}).
			WithLineStyle(
				NewLineStyle().
					WithColor("#c23531").
					WithWidth(2).
					WithType("solid").
					WithOpacity(0.8),
			),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	series := result["series"].([]interface{})
	lineSeries := series[0].(map[string]interface{})

	lineStyle := lineSeries["lineStyle"].(map[string]interface{})
	assert.Equal(t, "#c23531", lineStyle["color"])
	assert.Equal(t, float64(2), lineStyle["width"])
	assert.Equal(t, "solid", lineStyle["type"])
	assert.Equal(t, 0.8, lineStyle["opacity"])
}

func TestBarChartWithItemStyle(t *testing.T) {
	service := NewService()

	service.AddXAxis(
		NewAxis().
			WithType("category").
			WithData([]string{"Mon", "Tue", "Wed"}),
	)

	service.AddYAxis(NewAxis().WithType("value"))

	service.AddSeries(
		NewSeries().
			WithType("bar").
			WithData([]interface{}{120, 200, 150}).
			WithItemStyle(
				NewItemStyle().
					WithColor("#496E83").
					WithBorderColor("#fff").
					WithBorderWidth(1).
					WithBorderRadius([]interface{}{6, 6, 0, 0}).
					WithOpacity(0.9),
			),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	series := result["series"].([]interface{})
	barSeries := series[0].(map[string]interface{})

	itemStyle := barSeries["itemStyle"].(map[string]interface{})
	assert.Equal(t, "#496E83", itemStyle["color"])
	assert.Equal(t, "#fff", itemStyle["borderColor"])
	assert.Equal(t, float64(1), itemStyle["borderWidth"])
	assert.Equal(t, 0.9, itemStyle["opacity"])
}

func TestPieChartWithLabel(t *testing.T) {
	service := NewService()

	pieData := []map[string]interface{}{
		{"value": 335, "name": "Apple"},
		{"value": 310, "name": "Grapes"},
		{"value": 234, "name": "Pineapples"},
	}

	service.AddSeries(
		NewSeries().
			WithType("pie").
			WithData(pieData).
			WithLabel(
				NewLabel().
					WithShow(true).
					WithPosition("outside").
					WithFormatter("{b}: {c} ({d}%)"),
			).
			WithLabelLine(
				NewLabelLine().
					WithShow(true).
					WithSmooth(true),
			),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	series := result["series"].([]interface{})
	pieSeries := series[0].(map[string]interface{})

	label := pieSeries["label"].(map[string]interface{})
	assert.True(t, label["show"].(bool))
	assert.Equal(t, "outside", label["position"])
	assert.Equal(t, "{b}: {c} ({d}%)", label["formatter"])

	labelLine := pieSeries["labelLine"].(map[string]interface{})
	assert.True(t, labelLine["show"].(bool))
	assert.True(t, labelLine["smooth"].(bool))
}

func TestChartWithDataset(t *testing.T) {
	service := NewService()

	service.SetDataset(
		&Dataset{
			Source: [][]interface{}{
				{"Jan", 34, 20, 54},
				{"Feb", 28, 14, 64},
				{"Mar", 45, 32, 43},
			},
		},
	)

	service.AddXAxis(
		NewAxis().
			WithType("category").
			WithSplitLine(NewSplitLine().WithShow(false)),
	)

	service.AddYAxis(
		NewAxis().
			WithSplitLine(NewSplitLine().WithShow(false)),
	)

	service.AddSeries(
		NewSeries().
			WithName("series0").
			WithType("line"),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	assert.NotNil(t, result["dataset"])
	dataset := result["dataset"].(map[string]interface{})
	assert.NotNil(t, dataset["source"])
}

func TestEmptyChart(t *testing.T) {
	service := NewService()

	result, err := service.ToMap()
	require.NoError(t, err)

	assert.NotNil(t, result)
}

func TestChartWithMultipleAxes(t *testing.T) {
	service := NewService()

	service.AddXAxis(NewAxis().WithType("category").WithName("X1"))
	service.AddXAxis(NewAxis().WithType("value").WithName("X2"))

	service.AddYAxis(NewAxis().WithType("value").WithName("Y1"))
	service.AddYAxis(NewAxis().WithType("value").WithName("Y2"))

	service.AddSeries(
		NewSeries().
			WithType("line").
			WithData([]interface{}{1, 2, 3}).
			WithXAxisIndex(0).
			WithYAxisIndex(0),
	)

	service.AddSeries(
		NewSeries().
			WithType("bar").
			WithData([]interface{}{4, 5, 6}).
			WithXAxisIndex(1).
			WithYAxisIndex(1),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	xAxis := result["xAxis"].([]interface{})
	assert.Len(t, xAxis, 2)

	yAxis := result["yAxis"].([]interface{})
	assert.Len(t, yAxis, 2)

	series := result["series"].([]interface{})
	assert.Len(t, series, 2)

	lineSeries := series[0].(map[string]interface{})
	assert.Equal(t, float64(0), lineSeries["xAxisIndex"])
	assert.Equal(t, float64(0), lineSeries["yAxisIndex"])
}

func TestToJSON(t *testing.T) {
	service := NewService()

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddYAxis(NewAxis().WithType("value"))
	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	jsonData, err := service.ToJSON()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err)

	assert.NotNil(t, result["xAxis"])
	assert.NotNil(t, result["yAxis"])
	assert.NotNil(t, result["series"])
}

func TestBuildReturnsOption(t *testing.T) {
	service := NewService()

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddYAxis(NewAxis().WithType("value"))
	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	option := service.Build()
	assert.NotNil(t, option)
	assert.NotNil(t, option.XAxis)
	assert.NotNil(t, option.YAxis)
	assert.NotNil(t, option.Series)
}

func TestAxisWithSplitLine(t *testing.T) {
	service := NewService()

	service.AddXAxis(
		NewAxis().
			WithType("category").
			WithSplitLine(NewSplitLine().WithShow(false)),
	)

	service.AddYAxis(
		NewAxis().
			WithType("value").
			WithSplitLine(
				NewSplitLine().
					WithShow(true).
					WithLineStyle(
						NewLineStyle().
							WithColor("#e0e0e0").
							WithType("dashed"),
					),
			),
	)

	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	result, err := service.ToMap()
	require.NoError(t, err)

	xAxis := result["xAxis"].([]interface{})
	xSplitLine := xAxis[0].(map[string]interface{})["splitLine"].(map[string]interface{})
	assert.False(t, xSplitLine["show"].(bool))

	yAxis := result["yAxis"].([]interface{})
	ySplitLine := yAxis[0].(map[string]interface{})["splitLine"].(map[string]interface{})
	assert.True(t, ySplitLine["show"].(bool))
}

func TestAxisWithSplitArea(t *testing.T) {
	service := NewService()

	service.AddYAxis(
		NewAxis().
			WithType("value").
			WithSplitArea(
				NewSplitArea().
					WithShow(true),
			),
	)

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	result, err := service.ToMap()
	require.NoError(t, err)

	yAxis := result["yAxis"].([]interface{})
	ySplitArea := yAxis[0].(map[string]interface{})["splitArea"].(map[string]interface{})
	assert.True(t, ySplitArea["show"].(bool))
}

func TestSeriesWithStack(t *testing.T) {
	service := NewService()

	service.AddXAxis(
		NewAxis().
			WithType("category").
			WithData([]string{"Mon", "Tue", "Wed"}),
	)

	service.AddYAxis(NewAxis().WithType("value"))

	service.AddSeries(
		NewSeries().
			WithName("Income").
			WithType("bar").
			WithStack("Total").
			WithData([]interface{}{320, 302, 341}),
	)

	service.AddSeries(
		NewSeries().
			WithName("Expenses").
			WithType("bar").
			WithStack("Total").
			WithData([]interface{}{-120, -132, -101}),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	series := result["series"].([]interface{})

	incomeSeries := series[0].(map[string]interface{})
	assert.Equal(t, "Total", incomeSeries["stack"])

	expensesSeries := series[1].(map[string]interface{})
	assert.Equal(t, "Total", expensesSeries["stack"])
}

func TestPieChartWithCenterAndRadius(t *testing.T) {
	service := NewService()

	pieData := []map[string]interface{}{
		{"value": 52, "name": "XX"},
		{"value": 54, "name": "YY"},
		{"value": 42, "name": "ZZ"},
	}

	service.AddSeries(
		NewSeries().
			WithType("pie").
			WithCenter([]interface{}{"65%", 60}).
			WithRadius(35).
			WithData(pieData),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	series := result["series"].([]interface{})
	pieSeries := series[0].(map[string]interface{})

	center := pieSeries["center"].([]interface{})
	assert.Equal(t, "65%", center[0])
	assert.Equal(t, float64(60), center[1])

	assert.Equal(t, float64(35), pieSeries["radius"])
}

func TestLegendWithCustomPosition(t *testing.T) {
	service := NewService()

	service.SetLegend(
		NewLegend().
			WithShow(true).
			WithLeft("right").
			WithTop("top").
			WithOrient("vertical").
			WithData([]string{"Series A", "Series B"}),
	)

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddYAxis(NewAxis().WithType("value"))
	service.AddSeries(NewSeries().WithType("line").WithData([]interface{}{1, 2, 3}))

	result, err := service.ToMap()
	require.NoError(t, err)

	legend := result["legend"].(map[string]interface{})
	assert.Equal(t, "right", legend["left"])
	assert.Equal(t, "top", legend["top"])
	assert.Equal(t, "vertical", legend["orient"])

	legendDataInterface := legend["data"].([]interface{})
	legendData := make([]string, len(legendDataInterface))
	for i, v := range legendDataInterface {
		legendData[i] = v.(string)
	}
	assert.Equal(t, []string{"Series A", "Series B"}, legendData)
}

func TestSeriesWithEmphasis(t *testing.T) {
	service := NewService()

	service.AddXAxis(NewAxis().WithType("category"))
	service.AddYAxis(NewAxis().WithType("value"))

	service.AddSeries(
		NewSeries().
			WithType("line").
			WithData([]interface{}{1, 2, 3}).
			WithEmphasis(
				NewEmphasis().
					WithFocus("series").
					WithLabel(
						NewLabel().
							WithShow(true).
							WithFontSize(16),
					),
			),
	)

	result, err := service.ToMap()
	require.NoError(t, err)

	series := result["series"].([]interface{})
	lineSeries := series[0].(map[string]interface{})

	emphasis := lineSeries["emphasis"].(map[string]interface{})
	assert.Equal(t, "series", emphasis["focus"])

	label := emphasis["label"].(map[string]interface{})
	assert.True(t, label["show"].(bool))
	assert.Equal(t, float64(16), label["fontSize"])
}
