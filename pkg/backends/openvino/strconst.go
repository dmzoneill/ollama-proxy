package openvino

const (
	envModelPath = "OV_MODEL_PATH"

	openvinoGenerateScript = `
import json
import os
import sys
import openvino_genai as ov_genai

request = json.load(sys.stdin)
pipe = ov_genai.LLMPipeline(os.environ["OV_MODEL_PATH"], request["device"])

config = ov_genai.GenerationConfig()
config.max_new_tokens = request["max_tokens"]
config.temperature = request["temperature"]

result = pipe.generate(request["prompt"], config)
print(json.dumps({"response": result}))
`

	openvinoStreamScript = `
import json
import os
import sys
import openvino_genai as ov_genai

request = json.load(sys.stdin)
pipe = ov_genai.LLMPipeline(os.environ["OV_MODEL_PATH"], request["device"])

config = ov_genai.GenerationConfig()
config.max_new_tokens = request["max_tokens"]
config.temperature = request["temperature"]

class StreamPrinter(ov_genai.StreamerBase):
    def put(self, token_id):
        token = pipe.get_tokenizer().decode([token_id])
        print(json.dumps({"token": token, "done": False}), flush=True)
        return False

    def end(self):
        print(json.dumps({"token": "", "done": True}), flush=True)

streamer = StreamPrinter()
pipe.generate(request["prompt"], config, streamer)
`
)
