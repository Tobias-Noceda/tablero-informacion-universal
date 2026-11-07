import type { BoardPostitTemplate, PostitTemplate, WellKnown } from '$types/api';

// Post-it templates come after the whole catalogue in the picker.
export const TEMPLATE_ORDER = 1_000_000;

// A post-it template in the catalogue's shape, its one language in both, so
// every component that renders a well-known renders it too.
export function toWellKnown(template: PostitTemplate | BoardPostitTemplate): WellKnown {
	return {
		key: template.key,
		name_en: template.name,
		name_es: template.name,
		description_en: template.description || undefined,
		description_es: template.description || undefined,
		order: TEMPLATE_ORDER,
		title_en: template.title.text,
		title_es: template.title.text,
		resource: template.resource,
		request: template.request,
		response: template.response,
		query: template.query,
		rate: template.rate,
		parameters: (template.parameters ?? []).map((p) => ({
			key: p.key,
			label_en: p.label,
			label_es: p.label,
			type: p.type,
			placeholder_en: p.placeholder,
			placeholder_es: p.placeholder,
			default: p.default
		})),
		outputs: (template.outputs ?? []).map((o) => ({
			key: o.key,
			label_en: o.label,
			label_es: o.label,
			example_en: o.example,
			example_es: o.example
		})),
		builtin: false,
		updated_at: template.updated_at,
		updated_by: template.updated_by,
		scope: template.scope,
		usable: 'usable' in template ? template.usable : true
	};
}
