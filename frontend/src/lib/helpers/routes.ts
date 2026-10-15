import type { UUID } from '$types/api';

// Where an organization's boards are listed: "/" for Personal.
export function landingPath(orgId?: UUID) {
	return orgId ? (`/organizations/${orgId}` as const) : ('/' as const);
}

// A board under its organization, or at the top level when it is personal.
export function boardPath(id: UUID, orgId?: UUID) {
	return orgId ? (`/organizations/${orgId}/boards/${id}` as const) : (`/boards/${id}` as const);
}

// A card's detail, opened over its board.
export function nodePath(boardId: UUID, nodeId: UUID, orgId?: UUID) {
	return `${boardPath(boardId, orgId)}/nodes/${nodeId}` as const;
}
