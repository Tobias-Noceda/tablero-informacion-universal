import { headers } from "xmcp/headers";

// ApiError is a backend answer other than 2xx, with its `error` code.
// Its fields are declared, not parameter properties, so Node can run this
// file by stripping types (the tests do).
export class ApiError extends Error {
    readonly status: number;
    readonly code: string;
    // Which plan limit refused it (a 402 `plan_limit`), if any.
    readonly limit?: string;
    // The backend's explanation of a refused body (its `message`), if any.
    readonly detail?: string;

    constructor(status: number, code: string, limit?: string, detail?: string) {
        super(describe(status, code, limit, detail));
        this.name = "ApiError";
        this.status = status;
        this.code = code;
        this.limit = limit;
        this.detail = detail;
    }
}

// What the agent (and the person behind it) should read for each failure.
function describe(
    status: number,
    code: string,
    limit?: string,
    detail?: string,
) {
    switch (status) {
        case 400:
            return detail
                ? `Refused (${code}): ${detail}`
                : `The board service answered 400 (${code}).`;
        case 402:
            return limit === "api_access"
                ? "The MCP server requires a Pro or Max plan. Upgrade the account that created this access token."
                : `Plan limit reached (${limit ?? code}). Upgrade the plan of whoever owns the board, or remove something first.`;
        case 401:
            return "The access token is invalid, expired or revoked. Create a new one in your profile.";
        case 403:
            return code === "token_not_allowed"
                ? "Access tokens can only read and edit boards, their cards and card types."
                : `Not allowed (${code}).`;
        case 404:
            return "Not found, or you have no access to it.";
        default:
            return `The board service answered ${status} (${code}).`;
    }
}

function resolvePath(path: `/${string}`) {
    const url = process.env.API_URL || "http://localhost:31126";
    return new URL("/api" + path, url);
}

// The caller's bearer, read from the request being served: never kept in
// module state, which every concurrent client would share.
function authorization() {
    const value = headers().authorization;
    return Array.isArray(value) ? value[0] : value;
}

async function request(
    method: string,
    path: `/${string}`,
    body?: unknown,
): Promise<Response> {
    let response: Response;
    try {
        response = await fetch(resolvePath(path), {
            method,
            headers: {
                Authorization: authorization() ?? "",
                ...(body !== undefined && {
                    "Content-Type": "application/json",
                }),
            },
            body: body !== undefined ? JSON.stringify(body) : undefined,
        });
    } catch (err) {
        console.error("Network error:", err);
        throw new Error("Network error: unable to reach the board service");
    }

    if (!response.ok) {
        const payload = await response.json().catch(() => ({}));
        throw new ApiError(
            response.status,
            typeof payload?.error === "string" ? payload.error : "unknown",
            typeof payload?.limit === "string" ? payload.limit : undefined,
            typeof payload?.message === "string" ? payload.message : undefined,
        );
    }

    return response;
}

export const get = (path: `/${string}`) => request("GET", path);

export const post = (path: `/${string}`, body: unknown) =>
    request("POST", path, body);

export const put = (path: `/${string}`, body?: unknown) =>
    request("PUT", path, body);

export const patch = (path: `/${string}`, body: unknown) =>
    request("PATCH", path, body);

export const del = (path: `/${string}`, body?: unknown) =>
    request("DELETE", path, body);
