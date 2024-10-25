type FunctionCallType = {
	trans_id: string
	req_id: string
	result: string
	arguments: string
	tool_call_id: string
	function_name: string
	is_ok: boolean
}

export class FunctionCall {
  #originData: string
  #fc: FunctionCallType | null
  constructor(data: string) {
    this.#originData = data
    this.#fc = this.#parse()
  }
  get data() {
    return this.#fc
  }
  #parse() {
    try {
			const fc: FunctionCallType = JSON.parse(this.#originData)
			if (fc.tool_call_id && fc.req_id) {
				return fc
			}
			return null
		} catch (e) {
			return null
		}
  }
  readLLMArguments() {
    const args = this.#fc?.arguments
    if (!args) return null
    try {
      return JSON.parse(args)
    } catch (error) {
      return args
    }
  }
  writeLLMResult(result: string) {
    if (!this.#fc) return
    this.#fc.result = result
		this.#fc.is_ok = true
    return JSON.stringify(this.#fc)
  }
}