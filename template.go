package main

import "html/template"

var pageTmpl = template.Must(template.New("index").Parse(`
<!DOCTYPE html>
<html lang="fr">
<head>
<meta charset="UTF-8">
<title>mini-GED — démo Garage</title>
<style>
  body { font-family: -apple-system, sans-serif; max-width: 780px; margin: 40px auto; color: #222; }
  h1 { font-size: 1.4rem; }
  .flash { background: #eef7ee; border: 1px solid #8bc98b; padding: 10px 14px; border-radius: 6px; margin-bottom: 20px; }
  table { width: 100%; border-collapse: collapse; margin-top: 20px; }
  th, td { text-align: left; padding: 8px 6px; border-bottom: 1px solid #eee; font-size: 0.9rem; }
  form.inline { display: inline; }
  button { cursor: pointer; }
  .danger { color: #b00020; }
  .panel { border: 1px solid #ddd; border-radius: 8px; padding: 16px; margin-top: 24px; }
  .panel h2 { font-size: 1rem; margin-top: 0; }
  .badge { display: inline-block; background: #f5e6c8; color: #7a5c00; padding: 2px 8px; border-radius: 4px; font-size: 0.75rem; }
</style>
</head>
<body>

<h1>mini-GED — démo stockage objet (Garage)</h1>
<p>Bucket : <code>{{.Bucket}}</code></p>

{{if .Flash}}<div class="flash">{{.Flash}}</div>{{end}}

<form action="/upload" method="post" enctype="multipart/form-data">
  <input type="file" name="file" required>
  <button type="submit">Déposer le document</button>
</form>

<table>
  <tr><th>Nom</th><th>Taille</th><th>Modifié</th><th></th></tr>
  {{range .Docs}}
  <tr>
    <td>{{.Key}}</td>
    <td>{{.Size}} o</td>
    <td>{{.LastModified.Format "02/01/2006 15:04:05"}}</td>
    <td>
      <a href="/download?key={{.Key}}">télécharger</a>
      &nbsp;|&nbsp;
      <form class="inline" action="/delete" method="post" onsubmit="return confirm('Supprimer {{.Key}} ?');">
        <input type="hidden" name="key" value="{{.Key}}">
        <button class="danger" type="submit">supprimer</button>
      </form>
    </td>
  </tr>
  {{else}}
  <tr><td colspan="4">Aucun document pour l'instant.</td></tr>
  {{end}}
</table>

<div class="panel">
  <h2>Historique des versions <span class="badge">non disponible</span></h2>
  <p>
    Garage ne prend pas encore en charge le versioning natif des objets S3
    (fonctionnalité en discussion côté communauté, non livrée à ce jour).
    Il n'y a donc pas d'historique à afficher ici — à la différence de Ceph,
    qui gère nativement plusieurs versions par objet.
  </p>
  <form action="/overwrite-demo" method="post">
    <button type="submit">Démontrer l'écrasement (écrit 2 versions du même fichier)</button>
  </form>
</div>

</body>
</html>
`))
