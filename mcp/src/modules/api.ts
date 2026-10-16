import { current_user } from "./session";

function resolvePath(path: `/${string}`) {
    const url = process.env.API_URL || "http://localhost:31126";
    return new URL("/api" + path, url);
}

export async function get(
    path: `/${string}`,
    options?: RequestInit,
): Promise<Response> {
    const url = resolvePath(path);

    let response: Response;
    try {
        response = await fetch(new URL(url), {
            ...options,
            method: "GET",
            headers: {
                ...options?.headers,
                Authorization: `Bearer ${current_user}`,
            },
        });
    } catch (err) {
        console.error("Network error:", err);
        throw `Network error: Unable to reach server`;
    }

    if (!response.ok) {
        throw response.status;
    }

    return response;
}

export function post(
    path: `/${string}`,
    body: unknown,
    options?: RequestInit,
): Promise<Response> {
    const url = resolvePath(path);

    return fetch(url, {
        ...options,
        method: "POST",
        headers: {
            ...options?.headers,
            Authorization: `Bearer ${current_user}`,
            "Content-Type": "application/json",
        },
        body: JSON.stringify(body),
    });
}

export function put(
    path: `/${string}`,
    body?: unknown,
    options?: RequestInit,
): Promise<Response> {
    const url = resolvePath(path);

    return fetch(url, {
        ...options,
        method: "PUT",
        headers: {
            ...options?.headers,
            Authorization: `Bearer ${current_user}`,
            "Content-Type": "application/json",
        },
        body: JSON.stringify(body),
    });
}

export function patch(
    path: `/${string}`,
    body: unknown,
    options?: RequestInit,
): Promise<Response> {
    const url = resolvePath(path);

    return fetch(url, {
        ...options,
        method: "PATCH",
        headers: {
            ...options?.headers,
            Authorization: `Bearer ${current_user}`,
            "Content-Type": "application/json",
        },
        body: JSON.stringify(body),
    });
}

export function del(
    path: `/${string}`,
    body?: unknown,
    options?: RequestInit,
): Promise<Response> {
    const url = resolvePath(path);

    return fetch(url, {
        ...options,
        method: "DELETE",
        headers: {
            ...options?.headers,
            Authorization: `Bearer ${current_user}`,
        },
        body: body ? JSON.stringify(body) : undefined,
    });
}
