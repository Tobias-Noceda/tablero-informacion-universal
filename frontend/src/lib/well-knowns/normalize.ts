import type { BoardWellKnown, CustomWellKnown, WellKnown } from '$types/api';

// Custom card types come after the whole catalogue in the picker.
export const CUSTOM_ORDER = 1_000_000;

// A custom card type in the catalogue's shape, its one language in both, so
// every component that renders a well-known renders it too.
export function toWellKnown(custom: CustomWellKnown | BoardWellKnown): WellKnown {
	return {
		key: custom.key,
		name_en: custom.name,
		name_es: custom.name,
		description_en: custom.description || undefined,
		description_es: custom.description || undefined,
		order: CUSTOM_ORDER,
		title_en: custom.title.text,
		title_es: custom.title.text,
		resource: custom.resource,
		request: custom.request,
		response: custom.response,
		query: custom.query,
		rate: custom.rate,
		parameters: (custom.parameters ?? []).map((p) => ({
			key: p.key,
			label_en: p.label,
			label_es: p.label,
			type: p.type,
			placeholder_en: p.placeholder,
			placeholder_es: p.placeholder,
			default: p.default
		})),
		outputs: (custom.outputs ?? []).map((o) => ({
			key: o.key,
			label_en: o.label,
			label_es: o.label,
			example_en: o.example,
			example_es: o.example
		})),
		builtin: false,
		updated_at: custom.updated_at,
		updated_by: custom.updated_by,
		scope: custom.scope,
		usable: 'usable' in custom ? custom.usable : true
	};
}
