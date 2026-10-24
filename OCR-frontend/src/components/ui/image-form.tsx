import { Input } from "./input"
import { Label } from "./label"
import { Button } from "./button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "./card"
import LanguageCombobox from "./language-combobox"
import React from "react"
import {startProcessing, uploadImage} from "@/api/upload"
import { useToastManager } from "@/components/ui/toast"
import { useState } from 'react';

/*
type ImageFormProps = {
  setImg: (value: { image: File | null, text: string}) => void
  setProc: (value: boolean) => void
} */

/*
{
        contentType:image.type,
      }

*/


export default function ImageForm(){
  const toast = useToastManager()
  const [isSubmitting, setIsSubmitting] = useState(false)

  const [language, setLanguage] = React.useState("eng")
  const [image, setImage] = React.useState<File | null>(null)

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (isSubmitting || !image) return;

    setIsSubmitting(true)

    const toastId = toast.add({
      title: "Feltöltés folyamatban…",
      description: "Az objektum feltöltése folyamatban van.",
      type: "loading",
      timeout: 0,
    })

    try {
      const jobId = await uploadImage({contentType: image.type}, image);

      toast.update(toastId, {
        title: "Objektum feltöltése",
        description: "A feldolgozás indítása folyamatban van.",
        type: "loading",
        timeout: 0,
      })

      //Tell the backend that we're done
      await startProcessing(jobId)
      toast.update(toastId, {
        title: "A feldolgozás megkezdődött!",
        description: "A karakterfelismerés folyamatban van.",
        type: "success",
        timeout: 5000
      })

    } catch (error) {
      toast.update(toastId, {
        title: "Hiba történt!",
        description:
          error instanceof Error
            ? error.message
            : "Ismeretlen hiba történt.",
        type: "error",
        timeout: 5000
      })
      console.error("Image upload failed: ", error)
    } finally{
      setIsSubmitting(false)
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