### nayuki.cyou assets

#### File Structure:

```js
{
  "name": string,
  "description": string,
  "image": string,
  "button": {
    "text": string,
    "link": string
  },
  "status": IStatus
}

type IStatus = 'Active' | 'Inactive' | 'Archived';
```
