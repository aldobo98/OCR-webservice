import { Input } from "./input"
import { Label } from "./label"
import { Button } from "./button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "./card"
import React from "react"
import {startProcessing, uploadImage} from "@/api/upload"
import ComboBox from "./combobox"

type ImageFormProps = {
  setJobID: ( jobID:string ) => void;
}

export default function ImageForm({setJobID}: ImageFormProps){

  const docTypes = [
    {value: "structured", label: "Struktúrált"},
    {value: "unstructured", label: "Szöveges"}
  ]
  const [docType, setDocType] = React.useState("unstruct")
  const [image, setImage] = React.useState<File | null>(null)

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if(!image){
      return
    }
    try {
      const jobId = await uploadImage({contentType: image.type, documentType: docType}, image);
      console.log("Upload finished, JobID:", jobId);
      setJobID(jobId)

      //Tell the backend that we're done
      console.log("Calling startProcessing...");
      await startProcessing(jobId)
      console.log("startProcessing finished");
    } catch (error) {
      console.error("Image upload failed: ", error)
    }
  };

  return (
    <form onSubmit= {handleSubmit} >
    <Card className="w-full max-w-lg">
      <CardHeader>
        <CardTitle>Szövegfelismerés</CardTitle>
        <CardDescription>A szövegfelismeréshez egy dokumentumot feltölteni, és a szöveg típusát kell kiválasztani.</CardDescription>
      </CardHeader>
      <CardContent>
          <div className="flex flex-col pb-3">
            <Label>Dokumentum, ami szövegét kinyernéd:</Label>
            <Input type="file" id="image" required onChange={(e) => {
              if (e.target.files) {
                setImage(e.target.files[0])
              }
            }} />
          </div>
          <div>
            <Label>Dokumentum típusa:</Label>
            <ComboBox options={docTypes} value={docType} onChange={setDocType} placeholder="Válassz dokumentum típust" searchPlaceholder="Keress egy dokumentum típust" emptyMessage="Nincs ilyen dokumentum típus"></ComboBox>
          </div>
      </CardContent>
      <CardFooter className="flex-col gap-2">
        <Button type="submit">
          Szöveg felismerése
        </Button>
      </CardFooter>
    </Card>
    </form>
  )
}