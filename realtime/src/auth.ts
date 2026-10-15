export type BoardRole = "owner" | "editor" | "viewer";

export type Claims = { sub: string; exp: number };

export type RoleLookup = (
    token: string,
    board: string,
) => Promise<BoardRole | null>;

const boardRolesKnown = new Set<string>(["owner", "editor", "viewer"]);

// Reads a token's subject and expiry without checking its signature: they are
// only trusted once the platform API has accepted the same token.
export function claimsOf(token: string): Claims | null {
    const payload = token.split(".")[1];
    if (!payload) return null;

    try {
        const { sub, exp } = JSON.parse(
            Buffer.from(payload, "base64url").toString("utf8"),
        );
        if (typeof sub !== "string" || typeof exp !== "number") return null;
        return { sub, exp };
    } catch {
        return null;
    }
}

// The caller's role on a board, as `GET /boards/:id` answers it. Anything but
// a 200 with a known role means no access, an outage included.
export function boardRoles(api: string, timeoutMs = 5000): RoleLookup {
    return async (token, board) => {
        try {
            const res = await fetch(
                `${api}/boards/${encodeURIComponent(board)}`,
                {
                    headers: { Authorization: `Bearer ${token}` },
                    signal: AbortSignal.timeout(timeoutMs),
                },
            );
            if (res.status >= 500) {
                console.error("Board role lookup failed:", res.status);
            }
            if (res.status !== 200) {
                await res.body?.cancel();
                return null;
            }

            const { role } = (await res.json()) as { role?: unknown };
            return typeof role === "string" && boardRolesKnown.has(role)
                ? (role as BoardRole)
                : null;
        } catch (err) {
            console.error("Board role lookup failed:", err);
            return null;
        }
    };
}
