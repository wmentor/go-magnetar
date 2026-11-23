# Colab Demo

---

## Dependencies and Imports

---

```python
#@title Install dependencies

!pip install -q torchaudio omegaconf

import torch
from omegaconf import OmegaConf
from IPython.display import Audio, display


torch.hub.download_url_to_file('https://raw.githubusercontent.com/snakers4/silero-models/master/models.yml',
                               'latest_silero_models.yml',
                               progress=False)
models = OmegaConf.load('latest_silero_models.yml')
```

---

## List models

---

```python
# see latest avaiable models
available_models = models.denoise_models.models
print(f'Available models {available_models}')

for am in available_models:
    _models = list(models.denoise_models.get(am).keys())
    print(f'Available models for {am}: {_models}')
```

---

## SnS Latest

---

```python
import torch


name = 'small_slow' # 'large_fast', 'small_fast'
device = torch.device('cpu')

model, samples, utils = torch.hub.load(
  repo_or_dir='snakers4/silero-models',
  model='silero_denoise',
  name=name,
  device=device)

(read_audio, save_audio, denoise) = utils

model.to(device)  # gpu or cpu
```

---

### ex 0

---

```python
i = 0
torch.hub.download_url_to_file(
  samples[i],
  dst=f'sample{i}.wav',
  progress=True
)
audio_path = f'sample{i}.wav'
audio = read_audio(audio_path).to(device)
print('Original: ')
display(Audio(audio.cpu(), rate=24000))
output = model(audio)
save_audio(f'result{i}.wav', output.squeeze(1).cpu())
print('Saved: ')
display(Audio(f'result{i}.wav'))
print('Output: ')
display(Audio(output.squeeze(1).cpu(), rate=48000))
```

---

### ex 1

---

```python
i = 1
torch.hub.download_url_to_file(
  samples[i],
  dst=f'sample{i}.wav',
  progress=True
)
print('Original: ')
display(Audio(f'sample{i}.wav'))
output, sr = denoise(model, f'sample{i}.wav', f'result{i}.wav', device='cpu')
print('Saved: ')
display(Audio(f'result{i}.wav'))
print('Output: ')
display(Audio(output.cpu(), rate=sr))
```

---

# Minimal Example to Run Locally 

---

## Dependencies and Imports

---

```python
#@title Install dependencies

!pip install -q torch==2.0.0 torchaudio
```

---

## SnS Latest

---

```python
import os
import torch
import torchaudio


def read_audio(
    path: str,
    sampling_rate: int = 24000
):
    wav, sr = torchaudio.load(path)
    if wav.size(0) > 1:
        wav = wav.mean(dim=0, keepdim=True)
    if sr != sampling_rate:
        transform = torchaudio.transforms.Resample(
            orig_freq=sr,
            new_freq=sampling_rate
        )
        wav = transform(wav)
        sr = sampling_rate
    assert sr == sampling_rate
    return wav * 0.95


device = torch.device('cpu')
torch.set_num_threads(4)
local_file = 'model.pt'

if not os.path.isfile(local_file):
    torch.hub.download_url_to_file('https://models.silero.ai/denoise_models/sns_latest.jit',
                                   local_file)  

model = torch.jit.load(local_file)
torch._C._jit_set_profiling_mode(False) 
torch.set_grad_enabled(False)
model.to(device)

torch.hub.download_url_to_file('https://models.silero.ai/denoise_models/sample1.wav',
                                   dst=f'sample1.wav', progress=True)
audio_path = 'sample1.wav'
a = read_audio(audio_path)
a = a.to(device)
out = model(a)
print('Original: ')
display(Audio(a.cpu(), rate=24000))
print('Output: ')
display(Audio(out.cpu().squeeze(1).detach().numpy(), rate=48000))
```

---

```python

```
