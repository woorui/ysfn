import { Context, serve } from "../lib";
import * as jsonschema from "./jsonschema.json"

type Targs = {
	name: string;
};

async function handle(ctx: Context) {
	await sleep(1000 * 4);

	console.log("tag: " + ctx.tag + ", data: " + ctx.data);
	const args = ctx.readLLMArguments();
	// console.log(args);


	ctx.write(0x33, "resp" + ctx.data);
}

function sleep(ms: number) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

serve(
	[0xe001],
	handle,
	jsonschema,
);
