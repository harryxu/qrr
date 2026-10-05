package recipe

import "context"

// AskFunc supplies an interactive value using the parameter's current default.
type AskFunc func(context.Context, Param, any) (any, error)

// Prepare resolves parameters in declaration order and renders the recipe's argv.
// Options commands run only for omitted parameters that will be prompted.
// The recipe must have passed Parse or Validate. Explicit map entries override
// defaults even when false or empty. A nil ask disables all prompting.
func (r *Recipe) Prepare(ctx context.Context, explicit map[string]any, ask AskFunc) ([]string, error) {
	values := make(map[string]any, len(r.Params))
	for _, p := range r.Params {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		value, provided := explicit[p.Name]
		if !provided {
			value = DefaultValue(p)
			if ask != nil {
				var err error
				p, err = resolveOptions(ctx, p)
				if err != nil {
					return nil, err
				}
				value, err = ask(ctx, p, value)
				if err != nil {
					return nil, err
				}
			}
		}
		if err := ValidateValue(p, value); err != nil {
			return nil, err
		}
		values[p.Name] = value
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.Render(values)
}
