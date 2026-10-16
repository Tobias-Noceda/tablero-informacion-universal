import { type XmcpConfig } from "xmcp";

const config: XmcpConfig = {
  http: true,
  paths: {
    tools: "./src/tools",
    prompts: false,
    resources: false,
  },
  template: {
    icons: [{ src: "./xmcp.svg" }],
  }
};

export default config;
