import { apiKeyAuthMiddleware } from "xmcp";

export default [
    apiKeyAuthMiddleware({
        headerName: "Authorization",
        async validateApiKey(key) {
            const [schema, token] = key.split(" ", 2);
            return schema.toLowerCase() === "bearer" && !!token;
        },
    }),
];
