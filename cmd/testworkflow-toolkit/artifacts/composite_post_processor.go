package artifacts

type CompositePostProcessor struct {
	processors []PostProcessor
}

func NewCompositePostProcessor(processors ...PostProcessor) *CompositePostProcessor {
	return &CompositePostProcessor{processors: processors}
}

func (p *CompositePostProcessor) Start() error {
	for _, processor := range p.processors {
		if err := processor.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (p *CompositePostProcessor) Add(path string) error {
	for _, processor := range p.processors {
		if err := processor.Add(path); err != nil {
			return err
		}
	}
	return nil
}

func (p *CompositePostProcessor) End() error {
	for _, processor := range p.processors {
		if err := processor.End(); err != nil {
			return err
		}
	}
	return nil
}
