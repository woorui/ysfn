import net from "net";
import { Writable } from "stream";

export class Context {
	public tag: number;
	public data: string;
	private writer: Writable;
	private functionCall: FunctionCall | null;

	constructor(tag: number, data: string, writer: Writable) {
		this.tag = tag;
		this.data = data;
		this.writer = writer;
		this.functionCall = this.parseFunctionCall(data);
	}

	private parseFunctionCall(data: string): FunctionCall | null {
		try {
			const fc: FunctionCall = JSON.parse(data);
			if (fc.tool_call_id && fc.req_id) {
				return fc;
			}
			return null;
		} catch (e) {
			return null;
		}
	}

	public write(tag: number, data: string) {
		writeTagData(this.writer, tag, data);
	}

	public readLLMArguments() {
		const args = this.functionCall?.arguments;
		if (!args) {
			return null;
		}

		return JSON.parse(args);
	}

	public writeLLMResult(result: string) {
		if (!this.functionCall) {
			const fc = this.parseFunctionCall(this.data);
			if (fc) {
				this.functionCall = fc;
			} else {
				return;
			}
		}

		this.functionCall.result = result;
		this.functionCall.is_ok = true;
		this.write(REDUCE_TAG, JSON.stringify(this.functionCall));
	}
}

export function serve(
	tags: number[],
	handler: (ctx: Context) => Promise<void>,
	jsonschema: Object,
) {
	const title = JSON.stringify({
		tags: tags,
		function_definition: jsonschema,
	});

	const conn = net.createConnection(SOCK_PATH);

	conn.on("connect", () => {
		writeHeader(conn, title);

		conn.on("data", async (buf) => {
			while (buf.length >= 8) {
				const { tag, data } = readTagData(buf);

				const context = new Context(tag, data, conn);

				// TODO: timeout
				await handler(context);

				buf = buf.subarray(8 + data.length);
			}
		});
	});
}

function writeHeader(conn: net.Socket, title: string) {
	const len = Buffer.alloc(4);
	len.writeUInt32LE(title.length);

	conn.write(Buffer.concat([len, Buffer.from(title)]));
}

function readTagData(buf: Buffer) {
	const tag = buf.readUInt32LE(0);

	const length = buf.readUInt32LE(4);
	const data = buf.subarray(8, 8 + length).toString();

	return { tag, data: data };
}

function writeTagData(conn: Writable, tag: number, data: string) {
	const tagBuffer = Buffer.alloc(4);
	tagBuffer.writeUInt32LE(tag);

	const lenBuffer = Buffer.alloc(4);
	lenBuffer.writeUInt32LE(data.length);

	conn.write(Buffer.concat([tagBuffer, lenBuffer, Buffer.from(data)]));
}

const REDUCE_TAG = 0xe001;
const SOCK_PATH = "/Users/wurui/workspace/woorui/ysfn/sfn.sock";

type FunctionCall = {
	trans_id: string;
	req_id: string;
	result: string;
	arguments: string;
	tool_call_id: string;
	function_name: string;
	is_ok: boolean;
}
