import { MongoClient, type MongoClientOptions } from "mongodb";
import { attachDatabasePool } from "@vercel/functions";

const options: MongoClientOptions = {
    appName: "tesis.vercel.integration",
    maxIdleTimeMS: 5000,
};

const client = new MongoClient(process.env.MONGODB_URI!, options);
attachDatabasePool(client);

export default client;
