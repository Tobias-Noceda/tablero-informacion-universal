import { type XmcpConfig } from "xmcp";

const config: XmcpConfig = {
    // 0.0.0.0 so the ALB can reach it inside docker; every request still needs
    // a token (src/middleware.ts).
    http: { host: "0.0.0.0", port: 3001 },
    paths: {
        tools: "./src/tools",
        prompts: false,
        resources: false,
    },
    template: {
        icons: [{ src: "./xmcp.svg" }],
    },
};

export default config;
