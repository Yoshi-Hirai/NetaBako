"use client";

import { useState } from "react";
import styles from "./page.module.css";

export default function Home() {
	const [url, setUrl] = useState("");
	const [comment, setComment] = useState("");
	const [postText, setPostText] = useState("");

	return (
    	<div className={styles.page}>
    		<main className={styles.main}>
        		<h1>NetaBako</h1>
        		<p>投稿したいと思った瞬間を、そのまま投稿へ。</p>
				<label htmlFor="url">URL</label>
				<input
					id="url"
					type="url"
					placeholder="https://example.com"
					value={url}
					onChange={(event) => setUrl(event.target.value)}
				/>
				<label htmlFor="comment">どう思った？</label>
				<textarea
					id="comment"
					placeholder="ひとこと感想を入力"
					value={comment}
					onChange={(event) => setComment(event.target.value)}
				/>

				<button
					type="button"
					className={styles.primaryButton}
					onClick={() => setPostText(`${comment}\n\n${url}`)}
					disabled={!comment.trim()}
				>
				投稿文を作る
				</button>

				<label htmlFor="postText">投稿案</label>
				<textarea
					id="postText"
					value={postText}
					onChange={(event) => setPostText(event.target.value)}
				/>

				<button
				type="button"
				onClick={() => navigator.clipboard.writeText(postText)}
				disabled={!postText.trim()}
				>
				コピー
				</button>
			</main>
    	</div>
	);
}
