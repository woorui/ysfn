import { join } from 'path'
import * as net from 'net'
import { _genTools } from '@yomo/sfn'
import { Readable, Writable } from 'stream'
import { description, tags, handler } from './app'

const WORK_DIR = '.'
const FUNCTION_NAME = 'get_weather'
const SFN_FILE_PATH = 'app.ts'
const SOCK_PATH = join(WORK_DIR, 'debug.sock');

const REDUCE_TAG = 0xe001;

type FunctionCall = {
	trans_id: string;
	req_id: string;
	result: string;
	arguments: string;
	tool_call_id: string;
	function_name: string;
	is_ok: boolean;
}

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

function run() {
	if (!description || !tags || !handler || tags.length === 0) {
		throw Error('description, tags, handler signature must be exported!')
	}

	const tools = _genTools(FUNCTION_NAME, description, SFN_FILE_PATH)

	const title = JSON.stringify({
		tags: tags,
		function_definition: JSON.stringify(tools, null, 2)
	})

	const conn = net.createConnection(SOCK_PATH)

conn.on('connect', async () => {
		writeHeader(conn, title)

		while (true) {
			const [tag, data] = readTagData(conn)
			console.log(tag, data)
			const ctx = new Context(tag, data, conn)
			const args = ctx.readLLMArguments()

			const result = await handler(args)
			ctx.writeLLMResult(JSON.stringify(result))
		}
	})
}

function readTagData(stream: Readable): [number, string] {
	const tagBuf: Buffer = stream.read(4)
	const tag = tagBuf.readUInt32LE(0)

	const lengthBuf: Buffer = stream.read(4)
	const length = lengthBuf.readUInt32LE(0)

	const data = stream.read(length)

	return [tag, data]
}

function writeTagData(conn: Writable, tag: number, data: string) {
	const tagBuffer = Buffer.alloc(4);
	tagBuffer.writeUInt32LE(tag);

	const lenBuffer = Buffer.alloc(4);
	lenBuffer.writeUInt32LE(data.length);

	console.log(tag, data.length, data)

	conn.write(Buffer.concat([tagBuffer, lenBuffer, Buffer.from(data)]));
}

function writeHeader(conn: Writable, title: string) {
	const len = Buffer.alloc(4);
	len.writeUInt32LE(title.length);

	conn.write(Buffer.concat([len, Buffer.from(title)]));
}

run()
