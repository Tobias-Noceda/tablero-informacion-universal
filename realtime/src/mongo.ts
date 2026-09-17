import { MongoClient } from "mongodb";

export default new MongoClient(process.env.MONGODB_URI!, {
    appName: "tesis.vercel.integration",
});
