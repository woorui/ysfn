import net from 'net'

type Callbacks = {
  onData: (data: Buffer) => void
  onConnect: () => void
}

export function createConnection(
  sockPath: string,
  callbacks: Callbacks
) {
  if (!sockPath) throw Error('path is required!')
  const conn = net.createConnection(sockPath)
  conn.on('connect', () => {
    callbacks.onConnect()
  })
  conn.on('data', async (data: Buffer) => {
    await callbacks.onData(data)
  })
  return {
    conn,
  }
}