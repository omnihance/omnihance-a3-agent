package echarts

import (
	"encoding/json"
)

type Service struct {
	option *Option
}

func NewService() *Service {
	return &Service{
		option: &Option{},
	}
}

func (s *Service) Build() *Option {
	return s.option
}

func (s *Service) ToJSON() ([]byte, error) {
	return json.Marshal(s.option)
}

func (s *Service) ToMap() (map[string]interface{}, error) {
	data, err := json.Marshal(s.option)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) SetTitle(title *Title) *Service {
	s.option.Title = title
	return s
}

func (s *Service) SetTooltip(tooltip *Tooltip) *Service {
	s.option.Tooltip = tooltip
	return s
}

func (s *Service) SetLegend(legend *Legend) *Service {
	s.option.Legend = legend
	return s
}

func (s *Service) SetGrid(grid *Grid) *Service {
	s.option.Grid = grid
	return s
}

func (s *Service) AddXAxis(axis *Axis) *Service {
	if s.option.XAxis == nil {
		s.option.XAxis = make([]*Axis, 0)
	}
	s.option.XAxis = append(s.option.XAxis, axis)
	return s
}

func (s *Service) AddYAxis(axis *Axis) *Service {
	if s.option.YAxis == nil {
		s.option.YAxis = make([]*Axis, 0)
	}
	s.option.YAxis = append(s.option.YAxis, axis)
	return s
}

func (s *Service) AddSeries(series *Series) *Service {
	if s.option.Series == nil {
		s.option.Series = make([]*Series, 0)
	}
	s.option.Series = append(s.option.Series, series)
	return s
}

func (s *Service) SetBackgroundColor(color string) *Service {
	s.option.BackgroundColor = color
	return s
}

func (s *Service) SetColors(colors []string) *Service {
	s.option.Color = colors
	return s
}

func (s *Service) SetDataset(dataset *Dataset) *Service {
	s.option.Dataset = dataset
	return s
}

func (s *Service) SetAnimation(animation *bool) *Service {
	s.option.Animation = animation
	return s
}

func (s *Service) SetAnimationDuration(duration int) *Service {
	s.option.AnimationDuration = &duration
	return s
}

func (s *Service) SetAnimationEasing(easing string) *Service {
	s.option.AnimationEasing = easing
	return s
}
