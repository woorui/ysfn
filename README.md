# Yomo sfn nodejs wrapper

1. Link the `@yomo/sfn` package to global

```bash
cd yomo-sfn
pnpm run link
```

2. Compile the cli program
```bash
go build -o example/yrun
```

3. Write `example/.env`, the `.env` file is blow:
```
YOMO_SFN_NAME=get_weather
YOMO_SFN_ZIPPER=zipper.vivgrid.com:9000
YOMO_SFN_CREDENTIAL=
```

4. Run with the cli program
```bash
cd example
pnpm link --global @yomo/sfn
./yrun app.ts
```

5. Chat with vivgrid
```bash
curl https://openai.vivgrid.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${YOUR_TOKEN}" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "How is the weather in Pairs"}],
    "temperature": 0.7,
    "stream": true
  }'
```