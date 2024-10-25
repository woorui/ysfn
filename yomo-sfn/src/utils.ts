import { Writable } from 'stream'
import { generateSchema, getProgramFromFiles } from 'typescript-json-schema'

export function genTools(fnName: string, fnDesc: string, scriptPath: string) {
  const program = getProgramFromFiles([scriptPath])
  const parameters = generateSchema(program, 'Argument', { required: true })
  if (!parameters) {
    throw Error('Argument type must be defined!')
  }
  delete parameters['$schema']
  return {
      name: fnName,
      description: fnDesc,
      parameters,
  }
}

export function writeSFNHeader(writer: Writable, data: string) {
  const len = Buffer.alloc(4)
  len.writeUInt32LE(data.length)
  writer.write(Buffer.concat([len, Buffer.from(data)]))
}

export function readSFNData(buf: Buffer) {
  const tag = buf.readUInt32LE(0)
  const length = buf.readUInt32LE(4)
  const data = buf.subarray(8, 8 + length).toString()
  return { tag, data: data }
}

export function writeSFNData(writer: Writable, tag: number, data: string) {
  const tagBuffer = Buffer.alloc(4)
  tagBuffer.writeUInt32LE(tag)

  const lenBuffer = Buffer.alloc(4)
  lenBuffer.writeUInt32LE(data.length)

  writer.write(Buffer.concat([tagBuffer, lenBuffer, Buffer.from(data)]))
}

export function withTimeout(promise: Promise<any>, ms: number) {
  return Promise.race([
    promise,
		new Promise((_, reject) =>
			setTimeout(() => reject(new Error('Operation timed out')), ms)
		)
  ])
}