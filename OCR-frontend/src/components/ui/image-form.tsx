import { Input } from "./input"
import { Label } from "./label"
import { Button } from "./button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "./card"
import LanguageCombobox from "./language-combobox"
import React from "react"
import { getUploadUrl } from "@/api/upload"

/*
type ImageFormProps = {
  setImg: (value: { image: File | null, text: string}) => void
  setProc: (value: boolean) => void
} */


export default function ImageForm(){

  const [language, setLanguage] = React.useState("eng")
  const [image, setImage] = React.useState<File | null>(null)

  const handleSubmit = async (event: React.SubmitEvent) => {
    event.preventDefault();

    if (!image) {
      return;
    }

    try {
      const uploadUrl = await getUploadUrl({
        contentType:image.type,
      });
      console.log(uploadUrl)
    } catch (error) {
      console.error(error)
    }
  };

  return (
    <form onSubmit= {handleSubmit} >
    <Card className="w-full max-w-lg">
      <CardHeader>
        <CardTitle>Karakterfelismerés</CardTitle>
        <CardDescription>A karakterfelismeréshez egy szöveget tartalmazó képet kell feltölteni, és a képen látható szöveg nyelvét kell kiválasztani.</CardDescription>
      </CardHeader>
      <CardContent>
        
          <div className="flex flex-col pb-3">
            <Label>Kép, amelyről a szöveget kell felismerni:</Label>
            <Input type="file" id="image" required onChange={(e) => {
              if (e.target.files) {
                setImage(e.target.files[0])
              }
            }} />
          </div>
          <div>
            <Label>Képen látható szöveg nyelve:</Label>
            <LanguageCombobox value={language} onChange={setLanguage}></LanguageCombobox>
          </div>
      </CardContent>
      <CardFooter className="flex-col gap-2">
        <Button type="submit">
          Karakterek felismerése
        </Button>
      </CardFooter>
    </Card>
    </form>
  )
}