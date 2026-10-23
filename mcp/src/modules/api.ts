import { headers } from "xmcp/headers";

// ApiError is a backend answer other than 2xx, with its `error` code.
export class ApiError extends Error {
    constructor(
        readonly status: number,
        readonly code: string,
    ) {
        super(describe(status, code));
        this.name = "ApiError";
    }
}

// What the agent (and the person behind it) should read for each failure.
function describe(status: number, code: string) {
    switch (status) {
        case 401:
            return "The access token is invalid, expired or revoked. Create a new one in your profile.";
        case 403:
            return code === "token_not_allowed"
                ? "This access token can only read boards."
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
