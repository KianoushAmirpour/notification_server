package domain

const StoryGenerationPrompt = `
You are a story-generation assistant. Create a story based on the user's preferences by following below steps.

user preferences: %s

Instructions: 
1. Generate a roght plot:
Using user's input, produce a caopact outline that includes: Main characters, Setting, Central conflict, Key turning points, Ending direction (not fully detailed)
this plot should be up to 8 bullet points.

2. Expand into Scenes
Turn the plot into a scene-by-scene breakdown. For each scene, include: Purpose of the scene, What characters do or feel, Sensory or atmospheric notes and Important narrative developments.const
Include up to 6 scenes depending on story complexity.

3. Rewrite for Tone and Style
Turn the scene breakdown into a polished story written in the tone specified by the user.
Examples of tones: whimsical, dark and gritty, poetic, fast-paced thriller. 
You are allowed to use maximum 1000 words per story. 

4. Output the final story after fully integrating the tone/style requirements.

`

type Story struct {
	ID       int    `json:"id"`
	FileName string `json:"file_name"`
	UserID   int    `json:"user_id"`
	Story    string `json:"story"`
	Status   string `json:"status"`
}

type StoryRequestResponse struct {
	Message string `json:"message"`
}

type UploadStory struct {
	UserID int    `json:"user_id"`
	Story  string `json:"story"`
}
