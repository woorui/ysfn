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
