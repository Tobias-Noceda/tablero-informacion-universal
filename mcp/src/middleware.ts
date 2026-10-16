import { apiKeyAuthMiddleware } from "xmcp";

export default apiKeyAuthMiddleware({
  validateApiKey: () => Promise.resolve(true)
});
