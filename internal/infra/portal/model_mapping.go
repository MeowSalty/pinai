package portal

// mapModel 按当前映射规则转换模型名。
func (s *facadeService) mapModel(model string) (string, bool) {
	mappedModel, exists := s.modelMappingRule[model]
	if !exists {
		return model, false
	}

	return mappedModel, true
}
