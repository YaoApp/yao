package types

import jsoniter "github.com/json-iterator/go"

// GetOption returns the option for a provider
func (p *Provider) GetOption(id string) (*ProviderOption, bool) {
	if p.Options == nil {
		return nil, false
	}

	// Find the option by id
	for _, option := range p.Options {
		if option.Value == id {
			return option, true
		}
	}

	return nil, false
}

// GetOptionByIndex returns the option by index
func (p *Provider) GetOptionByIndex(index int) (*ProviderOption, bool) {
	if len(p.Options) <= index {
		return nil, false
	}
	return p.Options[index], true
}

// Parse parses the provider option
func (p *ProviderOption) Parse(v interface{}) error {

	raw, err := jsoniter.Marshal(p)
	if err != nil {
		return err
	}

	err = jsoniter.Unmarshal(raw, v)
	if err != nil {
		return err
	}

	return nil
}
