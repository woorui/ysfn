import { Context, serve } from "../lib";

type Args = {
    order_id: string
};

// curl --request POST \
//   --url http://localhost:8000/v1/chat/completions \
//   --header 'Authorization: Bearer ZAjwlwZqeGZiDAh|ZvVVtmVuZWNGbJevrYmcLGiAPCceop' \
//   --header 'Content-Type: application/json' \
//   --data '{
//   "model": "gpt-4o",
//   "messages": [
//     {
//       "role": "system",
//       "content": "You are a helpful customer support assistant. Use the supplied tools to assist the user."
//     },
//     {
//       "role": "user",
//       "content": "Hi, can you tell me the delivery date for my order?"
//     },
//     {
//       "role": "assistant",
//       "content": "Hi there! I can help with that. Can you please provide your order ID?"
//     },
//     {
//       "role": "user",
//       "content": "i think it is order_12345"
//     }
//   ],
//   "stream": false
// }'
async function handle(ctx: Context) {
    console.log("tag: " + ctx.tag + ", data: " + ctx.data);

    const args: Args = ctx.readLLMArguments();
    console.log("order_id: " + args.order_id);

    ctx.writeLLMResult(`{"order_id": "${args.order_id}", "delivery_date": "2024-08-30"}`);
}

serve([0x33], handle);
